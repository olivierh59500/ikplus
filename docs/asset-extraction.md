# Preparing the original game data

The playable build includes the generated IK+ resources, while the original ADF
is not distributed. The included Python tools can reproduce the graphics,
audio, gameplay tables and metadata from a compatible disk.

## Supported disk

The verified input is:

```text
IK+ (1988-12-20)(System 3)(de-en)[cr][h Sign].adf
```

The uncompressed file is 901,120 bytes and has SHA-256:

```text
d1662b6b1704ca211d0cd0fd9899a9b6e92b01a892c7766110e5730ddee3e266
```

The [Planet Emulation Amiga ADF catalogue, letter I](https://www.planetemu.net/roms/commodore-amiga-games-adf?page=I)
can be used to identify the revision. Extract the ADF from its archive before
running the tools.

## Generate the resources

Place the disk at:

```text
.cache/reference/disks/IK+ (1988-12-20)(System 3)(de-en)[cr][h Sign].adf
```

Then run:

```sh
make assets
make verify-assets
make build
```

An explicit input and output can also be supplied to the first extraction step:

```sh
python3 tools/extract_adf.py --adf "/path/to/IK+.adf" --output .cache/rebuild
```

The extractor validates the Amiga OFS chains, disk checksum and ByteKiller
stream before producing data. The remaining scripts regenerate the native
random, audio, graphics, gameplay and shadow tables. The ADF and intermediate
program data remain local.
