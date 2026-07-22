## Docs gen

This utility is use for generating a mintlify docs.json file from multiple yaml files.
Website common settings (eg colors, anchor links) should be in a common.yaml file and each tab/version of the docs should be in it's own file with

```
navigation:
  tabs:
    - tab: Section
```

To generate the docs.json file you pass all yaml files to the command or run `make docs.json` from the root of this repo.

```bash
docs-gen common.yaml \
  talos-1.11.yaml \
  talos-1.10.yaml \
  omni.yaml \
  kubernetes.yaml
```
