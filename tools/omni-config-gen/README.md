## Omni Config Gen

This utility generates the Omni configuration reference documentation page from the Omni JSON schema.

It reads the schema, extracts all configuration fields with their types, descriptions, default values, CLI flags, and constraints, and outputs a single MDX file with HTML tables.

### Usage

Pass the path or URL to the Omni configuration schema:

```bash
# From a local file
go run . /path/to/schema.json > ../public/omni/reference/omni-configuration.mdx

# From a URL (default in Makefile)
go run . https://raw.githubusercontent.com/siderolabs/omni/refs/heads/main/internal/pkg/config/schema.json > ../public/omni/reference/omni-configuration.mdx
```

Or use the Makefile target from the repo root:

```bash
make generate-omni-config-reference
```

To use a local schema file instead of the default URL:

```bash
make generate-omni-config-reference OMNI_CONFIG_SCHEMA_URL=/path/to/schema.json
```
