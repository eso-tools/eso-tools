# The Elder Scrolls Online tools

A set of libraries and tools for The Elder Scrolls Online

## Downloads

Download the compiled binaries in [Releases](https://github.com/eso-tools/eso-tools/releases)

## Usage

### MNF extracter

Extract all files from a .mnf file:

```powershell
mnf-extracter `
    extractAll `
    --input "C:\Program Files (x86)\Zenimax Online\The Elder Scrolls Online\game\client\game.mnf" `
    --output ".\game-data" `
    --threads 3
```

Extract specific file from a .mnf file:

```powershell
mnf-extracter `
    extractFile `
    --input "C:\Program Files (x86)\Zenimax Online\The Elder Scrolls Online\depot\eso.mnf" `
    --output ".\eso-data" `
    --id "0x01000012-00000000"
```

Dump a .mnf file to .csv:

```powershell
mnf-extracter `
    dumpMnf `
    --input "C:\Program Files (x86)\Zenimax Online\The Elder Scrolls Online\game\client\game.mnf" `
    --output ".\game.csv"
```

Parse a .lang file to .csv:

```powershell
mnf-extracter `
    parseLng `
    --input ".\en.lang" `
    --output ".\en-lang-data"
```

Write a .lang file from .csv:

```powershell
mnf-extracter `
    writeLng `
    --input ".\en-lang-data" `
    --output ".\en.lang"
```
