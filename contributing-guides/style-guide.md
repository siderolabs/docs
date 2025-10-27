# Documentation style guide

To ensure consistency and cohesiveness in the SideroLabs documentation, here are the rules to follow when creating new documentation:

## 1) Lists & Procedures

Use a step-by-step (numbered) list for actions you want a user to take (download iso, boot the machine, etc.).

- Each step = one action.
- Use short sub-bullets for options or notes under a step.

Use an unordered ****list for non-sequential items (features, tips, caveats).

**Examples**

- ✅ Numbered (sequential):
    1. Download the ISO.
    2. Boot the machine.
    3. Select **Install**.
- ✅ Unordered (non-sequential):
    - Troubleshooting tips
    - Known limitations
    - Related links

## 2) Headings

- **No stacked headings:** Don’t put a heading immediately after another heading. Add at least one sentence before the next heading.
- **Hierarchy:** Do not jump backwards when using headers. Follow this sequence: Page title→ `##` → `###`not `###` then `##`.
- **Keep it progressive:** Move down one level at a time as you drill into detail.
- Use sentence case.

## 3) Referencing UI Text

- Refer to UI elements in **bold**: **Settings**, **Save**, **Create cluster**.
- Use **bold** with chevrons for paths: **Settings** > **Networking** > **Add**.

## 4) Code & Commands

- Use fenced code blocks with a language hint (e.g ```bash, ```json, etc,).
- **Do not** include the shell prompt (`$` or `#`) in copy-pasteable commands.
- For long commands, use line continuations (`\`) or split into multiple blocks.
- Export variables if they will be reused to avoid mistyping, and put all exported variables as the first step whenever possible for easier reference later.
- Don't use commands that have different behaviour between BSD and GNU (e.g., `sed`).
- Use placeholders in `UPPER_SNAKE_CASE` or `<angle-brackets>` and define them above the block.
- Always put placeholders that needs to be replaced at the end of the command (e.g., `export var=<my_var>`)

**Good**

```bash

kubectl apply -f deployment.yaml
```

**Bad**

```bash

$ kubectl apply -f deployment.yaml
```

## 5) Links

- Use descriptive link text: “See the **Cluster templates guide**” (not “click here”).

## 6) Images & Screenshots

**Filenames**

- Use **kebab-case**, descriptive names; no spaces or auto-generated names.
- Example: `accessing-exposed-service.png` (✅) vs `Screenshot 123344 (1) (1).png` (❌).

**Content & Privacy**

- **Remove/blur personal data** (emails, IDs, internal URLs, API keys).
- Prefer neutral/demo data. You can use consistent neutral/demo data with themed names for better readability. Examples of these names are:
    - **Classic Names**: Alice Cooper, Bob Smith, Charlie Davis, Diana Prince, Eve Johnson.
    - **Fantasy**: Frodo Baggins, Hermione Granger, Luke Skywalker, Arya Stark.
    - **Companies**: Acme Corp, Globex Industries, Wayne Enterprises, Stark Industries, Umbrella Corp.
    - **Emails**: Use `@example.com`, `@company.local`, or `@demo.org` domains.
- Ensure readability in both light and dark modes if applicable.

## 7) Voice & Style

- Be direct and action-oriented (“Click **Save**”, “Run the command”).
- Use present tense and active voice.
- Define acronyms on first use.
- Prefer simple language; avoid jargon unless necessary (and define it).