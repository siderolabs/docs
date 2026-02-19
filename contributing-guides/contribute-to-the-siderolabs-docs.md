# How to contribute to the SideroLabs documentation

This guide walks you through how to make changes to the SideroLabs documentation.

Before you begin, please read our [Contributor Guide](../README.md).

It explains how the documentation is organized, how issues are labeled, and which style guide to follow when writing or editing content.

## Step 1: Prerequisites

Before contributing to the SideroLabs documentation, make sure you have the following tools installed:

* Git – for cloning repositories and managing your changes
* Make – for running local build and validation commands
* Docker – for previewing the documentation site locally

> **Note for macOS users**: macOS ships BSD make by default, not GNU make. Install GNU make with `brew install make` and use `gmake` instead of `make` if you encounter build errors.

If you’re not sure whether these tools are already installed, you can verify by running the following commands in your terminal:

```bash
git --version
make --version
docker --version
```

If any of these return an error, you’ll need to install them before proceeding. You can find installation instructions for your operating system here:

* [Install Git](https://git-scm.com/downloads)
* [Install Make](https://www.gnu.org/software/make/)
* [Install Docker](https://docs.docker.com/get-started/get-docker/)

Once everything is set up, you’re ready to clone the repository and start contributing.

## Step 2:  Fork and clone the repository

All SideroLabs documentation lives in the main docs repository.

To make contributions, you’ll first need to fork the repository and clone it locally:

1. Go to the [SideroLabs Docs repository.](https://github.com/siderolabs/docs).

2. Click **Fork** in the top right corner to create your own copy.

![Fork icon in the SideroLabs Docs repository](../public/images/contribute-to-the-siderolabs-fork.png)

3. Clone your fork to your local machine.

    ```bash
    git clone https://github.com/<your-username>/docs.git
    ```

4. Create a new branch for your changes:	

    ```bash
    cd docs
    git checkout -b update-docs-topic
    ```

Now that you’re inside the cloned repository, run the following command to verify your setup and see all available tools for working with the docs; including how to preview pages, check for broken links, and catch missing files before pushing changes:

```bash
make help
```

## Step 3: Make a change

This section walks you through how to create and validate documentation, whether you're editing an existing page or adding a new one.

### Edit an existing page

To update an existing file:

1. Locate and modify the file with your desired changes.
2. Preview your work locally:

    ```bash
    make preview
    ```

3. Check for broken links:

    ```bash
    make broken-links
    ```

4. Verify that your new document follows the SideroLabs documentation style guide by running:

    ```bash
    make vale DOC=<link-to-the-doc-addition>
    ```

### Add a New Page

To add a new page to the docs:

1. Create a new MDX file in the appropriate folder:
    * Talos (versioned): `public/talos/&lt;version>/&lt;section>/your-page.mdx`
    * Omni: `public/omni/&lt;section>/your-page.mdx`
    * Kubernetes Guides: `public/kubernetes-guides/&lt;section>/your-page.mdx`
2. Add the new page to the sidebar. Mintlify doesn’t automatically detect new pages.

    To make your new page visible in the sidebar, add the file path of the new page to the correct YAML file:

    * Talos → `talos-v&lt;version>.yaml`
    * Omni → `omni.yaml`
    * Kubernetes Guides → `kubernetes-guides.yaml`
3. Once your YAML is updated, rebuild the master `docs.json` file by running:

    ```bash
    make docs.json
    ```

	> This step compiles all the YAML sidebar files into one `docs.json` file that Mintlify uses to render navigation.

4. Preview your work locally:

    ```bash
    make preview
    ```

5. Check for broken links:

    ```bash
    make broken-links
    ```

6. Verify that your new document follows the [SideroLabs documentation style guide](./style-guide.md) by running:

    ```bash
    make vale DOC=<link-to-the-doc-addition>
    ```

## Step 4: Commit and Push

Once you have verified that everything looks good, commit your changes:

```bash
git add .
git commit -m "docs: improve &lt;topic> section"
git push -u origin HEAD
```

## Step 5: Open a Pull Request

Finally open a pull request:

1. Go to your fork on GitHub and click Compare & Pull Request.
2. In the PR description:
    * Explain what you changed.
    * Describe why it’s needed.
    * Add screenshots if your changes affect structure or layout.
3. Link any related issues (e.g., Closes #123).

Once reviewed and merged, your contribution will automatically appear in the live docs on the next deployment.

Thank you for helping improve the SideroLabs documentation!
