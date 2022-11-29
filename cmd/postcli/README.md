# postcli

CLI tool for PoST initialization

# Build

```bash
git clone https://github.com/spacemeshos/post
cd post
make postcli
```

# Usage

```bash
cd build
./postcli --help
```

###  Print the list of compute providers

```bash
./postcli -printProviders
```

### Print the used config and options

```bash
./postcli -printConfig
```

### Initialize 

Example

```bash
./postcli -provider=2 -id=c230c51669d1fcd35860131e438e234726b2bd5f9adbbd91bd88a718e7e98ecb -commitmentAtxId=c230c51669d1fcd35860131e438e234726b2bd5f9adbbd91bd88a718e7e98ecb
 
```

### Remarks
* `-id` and `-commitmentAtxId` are required because they are committed to the generated data.
* If `-id` isn't provided, the id (public key) will be auto-generated, while saving `key.bin` in `-datadir`.
* If `postcli` is called multiple times on a given `-datadir`, config mismatch error is likely to occur. In this case, the `-reset` flag can be used to easily clean the previous instance. 
