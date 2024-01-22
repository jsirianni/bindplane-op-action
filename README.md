# bindplane-op-action

## Usage

### Export Resources

```bash
bindplane get destination -o yaml --export > destination.yaml
bindplane get configuration -o yaml --export > configuration.yaml
```

### Workflow

The following workflow can be used as an example.

```yaml
name: bindplane

on:
  push:
    branches:
      - main

# Write back requires access to the repo
permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Run GoReleaser
        uses: jsirianni/bindplane-op-action@main
        with:
          remote_url: ${{ secrets.BINDPLANE_REMOTE_URL }}
          username: ${{ secrets.BINDPLANE_USERNAME }}
          password: ${{ secrets.BINDPLANE_PASSWORD }}
          api_key: "" # Optional replacement for username and password
          destination_path: test/resources/destinations/resource.yaml
          configuration_path: test/resources/configurations/resource.yaml
          # Write back requires these three options
          configuration_output_dir: test/otel/${{ matrix.bindplane_versions }}
          configuration_output_branch: dev
          token: ${{ secrets.GITHUB_TOKEN }}
```
