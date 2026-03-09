# Amanda: Implement Galaxy Proxy

## Purpose

This feature implements Galaxy Proxy support into Amanda. This feature will allow Amanda to be able to proxy requests for a collection which it doesn't have stored in the `artifacts` directory locally, and go to a Galaxy compatible API server and pull the collection from the upstream Galaxy server, store it and serve it to the `ansible-galaxy` client.

If you have any doubts or questions about any details of this plan, ask clarifying questions.

## CLI Flags

The implementation should have two CLI flags:

1. `galaxy-proxy` - A boolean flag which enables the proxy functionality.
2. `galaxy-host` - A string option which allows a user to specify a custom Galaxy upstream server. The default should be `https://galaxy.ansible.com`.

## Scenarios

The following scenarios should be covered by this Galaxy Proxy feature. This feature MUST support these scenarios.

In both of the following scenarios, if the collection cannot be found or the specified version cannot be found on the Galaxy upstream server then Amanda should respond with a HTTP 404 (Not Found). If there is an error in the download process, then Amanda should respond with a HTTP 500 (Internal Server Error). If the collection SHA256 verification process fails, then Amanda should respond with a HTTP 500 (Internal Server Error).

### Scenario 1

An `ansible-galaxy` client can request a collection to install, for example `community.general`:

```
ansible-galaxy collection install community.general --force
```

When the Galaxy proxy feature is enabled, for scenario 1 the flow should go like this:

1. Check that the collection exists on the Galaxy upstream server using the [Galaxy NG version 3 API - Get a specific collection](https://docs.ansible.com/projects/galaxy-ng/en/latest/community/api_v3.html#get-a-specific-collection) endpoint.

2. If the collection exists, check what the newest available version of the collection is. Using the [Galaxy NG version 3 API - List collection versions](https://docs.ansible.com/projects/galaxy-ng/en/latest/community/api_v3.html#list-collection-versions) endpoint.

3. Based on the data returned by 2, check whether we have that latest version of the collection available already locally. If we do then serve that directly to the client.

4. If the latest version is not available locally, download the collection from the Galaxy upstream server and store it locally. Use the [Galaxy NG version 3 API - Get a specific collection version](https://docs.ansible.com/projects/galaxy-ng/en/latest/community/api_v3.html#get-a-specific-collection-version) endpoint. During this download process the SHA256 hash of the collection `tar.gz` file should be validated before storing it and serving it to the client.

### Scenario 2

An `ansible-galaxy` client can request a specific version of a collection to install, for example `community.general` version `12.3.0`:

```
ansible-galaxy collection install community.general:==12.3.0 --force
```

When the Galaxy proxy feature is enabled, for scenario 2 the flow should go like this:

1. Check that the collection exists on the Galaxy upstream server using the [Galaxy NG version 3 API - Get a specific collection](https://docs.ansible.com/projects/galaxy-ng/en/latest/community/api_v3.html#get-a-specific-collection) endpoint.

2. If the collection version exists, check whether we have that specific version of the collection already available locally and serve that to the client.

3. If the collection version does not exist locally, check whether that version is available on the Galaxy upstream server using the [Galaxy NG version 3 API - Get a specific collection version](https://docs.ansible.com/projects/galaxy-ng/en/latest/community/api_v3.html#get-a-specific-collection-version) endpoint.

4. If the collection version is available on the Galaxy upstream server, then download the collection and store it locally. During this download process the SHA256 hash of the collection `tar.gz` file should be validated before storing it and serving it to the client.

## Useful Documentation

- Galaxy NG version 3 API - Get a specific collection: https://docs.ansible.com/projects/galaxy-ng/en/latest/community/api_v3.html#get-a-specific-collection

- Galaxy NG version 3 API - List collection versions: https://docs.ansible.com/projects/galaxy-ng/en/latest/community/api_v3.html#list-collection-versions

- Galaxy NG version 3 API - Get a specific collection version: https://docs.ansible.com/projects/galaxy-ng/en/latest/community/api_v3.html#get-a-specific-collection-version
