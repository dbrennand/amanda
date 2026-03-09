// (c) Daniel Brennand <contact@danielbrennand.com>
// GNU General Public License v3.0+
//     (see https://www.gnu.org/licenses/gpl-3.0.txt)

package galaxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/sivel/amanda/storage"
)

var ErrNotFound = errors.New("not found on upstream")

type upstreamCollection struct {
	HighestVersion struct {
		Version string `json:"version"`
	} `json:"highest_version"`
}

type upstreamVersion struct {
	DownloadURL string `json:"download_url"`
	Artifact    struct {
		SHA256 string `json:"sha256"`
	} `json:"artifact"`
}

type inflight struct {
	once sync.Once
	err  error
}

type Galaxy struct {
	host     string
	storage  *storage.Storage
	client   *http.Client
	inflight sync.Map
}

func New(host string, storage *storage.Storage) *Galaxy {
	return &Galaxy{
		host:    host,
		storage: storage,
		client:  &http.Client{},
	}
}

func (g *Galaxy) collectionURL(namespace, name string) string {
	return fmt.Sprintf(
		"%s/api/v3/plugin/ansible/content/published/collections/index/%s/%s/",
		g.host, namespace, name,
	)
}

func (g *Galaxy) versionURL(namespace, name, version string) string {
	return fmt.Sprintf(
		"%s/api/v3/plugin/ansible/content/published/collections/index/%s/%s/versions/%s/",
		g.host, namespace, name, version,
	)
}

func (g *Galaxy) get(url string, target interface{}) error {
	resp, err := g.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream returned %d for %s", resp.StatusCode, url)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func (g *Galaxy) filename(namespace, name, version string) string {
	return fmt.Sprintf("%s-%s-%s.tar.gz", namespace, name, version)
}

func (g *Galaxy) Fetch(namespace, name, version string) error {
	if version != "" {
		// Specific version: check cache first
		if g.storage.Exists(g.filename(namespace, name, version)) {
			log.Printf("galaxy: %s.%s:%s already cached", namespace, name, version)
			return nil
		}
	}

	// Resolve version if needed
	var col upstreamCollection
	if err := g.get(g.collectionURL(namespace, name), &col); err != nil {
		return err
	}

	if version == "" {
		version = col.HighestVersion.Version
		if version == "" {
			return ErrNotFound
		}
		log.Printf("galaxy: %s.%s resolved latest version %s", namespace, name, version)
		// Check cache after resolving latest version
		if g.storage.Exists(g.filename(namespace, name, version)) {
			log.Printf("galaxy: %s.%s:%s already cached", namespace, name, version)
			return nil
		}
	}

	// Dedup concurrent downloads for the same file
	key := g.filename(namespace, name, version)
	actual, _ := g.inflight.LoadOrStore(key, &inflight{})
	entry := actual.(*inflight)
	entry.once.Do(func() {
		entry.err = g.download(namespace, name, version)
		g.inflight.Delete(key)
	})
	return entry.err
}

func (g *Galaxy) download(namespace, name, version string) error {
	log.Printf("galaxy: downloading %s.%s:%s from upstream", namespace, name, version)
	var ver upstreamVersion
	if err := g.get(g.versionURL(namespace, name, version), &ver); err != nil {
		return err
	}

	if ver.DownloadURL == "" {
		return fmt.Errorf("upstream returned empty download URL")
	}

	resp, err := g.client.Get(ver.DownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream download returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Printf("galaxy: verifying SHA256 for %s.%s:%s", namespace, name, version)
	_, err = g.storage.Write(ver.Artifact.SHA256, g.filename(namespace, name, version), bytes.NewReader(body))
	return err
}
