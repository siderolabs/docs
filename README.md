## Sidero docs

Repo for docs managed at [docs.siderolabs.com](https://docs.siderolabs.com)

## Writing docs

Writing docs from this repo requires `docker`, and `make` installed.
You should be able to run

```bash
make preview
```

and start writing in your editor of choice.

If you are adding new sections or pages you will need to update the relevant yaml file(s) and run

```
make docs.json common.yaml talos-1.10.yaml ... omni.yaml
```

This file is needed for mintlify to preview the docs locally and to publish docs.
The document structure is ordered in the order you provide the .yaml files.

The generated json file is automatically verified against the mintlify published schema.

## Directory structure

Documents are structured in folders based on the website tabs, talos, omni, kubernetes.
If the documentation has versioning it should have a sub folder for each version (eg talos/v1.11).

Within each folder is a top level section (eg Getting Started).
Each section should have it's own `/images/` folder for images used in that section.

Sections can also have subfolders for documentation that has multiple parts.
Subfolders should not have their own `/images/` folder.
