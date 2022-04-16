# The Elder Scrolls Online tools

A set of libraries and tools for The Elder Scrolls Online

## Downloads

Download the compiled binaries in [Releases](https://github.com/eso-tools/eso-tools/releases)

## Usage

### MNF extracter

Extract all files from a .mnf file:

```powershell
.\mnf-extracter.exe `
    extractAll `
    --input "C:\Program Files (x86)\Zenimax Online\The Elder Scrolls Online\game\client\game.mnf" `
    --output "./game-data" `
    --threads 3
```

Dump a .mnf file to .csv:

```powershell
.\mnf-extracter.exe `
    dumpMnf `
    --input "C:\Program Files (x86)\Zenimax Online\The Elder Scrolls Online\game\client\game.mnf" `
    --output "./game.csv"
```
