# postcli

CLI tool for PoST initialization

## Getting it

Go to the <https://github.com/spacemeshos/post/releases> and take the most recent release for your platform. In case if you want to build it from source, follow the instructions below.

```bash
git clone https://github.com/spacemeshos/post
cd post
make postcli
```

## Usage

```bash
./postcli --help
```

## Get OpenCL working

You need to have OpenCL support on your system. OpenCL usually comes with your graphics drivers. On Windows it should work out of the box on linux you will need to install them separately.

You can always list the providers by using

```bash
clinfo -l
```

That's separate command NOT shipped with post implementation. Please refer to your system installation manual of clinfo for installation instructions.

### Nvidia

```bash
apt install nvidia-opencl-icd
```

### AMD

<https://amdgpu-install.readthedocs.io/en/latest/install-script.html>

### Intel

<https://github.com/intel/compute-runtime/releases>

## Print the list of compute providers

```bash
./postcli -printProviders
```

## Print the number of files that would be initialized

```bash
./postcli -numUnits 100 -printNumFiles
1600
```

## Print the used config and options

```bash
./postcli -printConfig
```

## Initializing PoST data

Example

```bash
./postcli -provider=2 -id=c230c51669d1fcd35860131e438e234726b2bd5f9adbbd91bd88a718e7e98ecb -commitmentAtxId=c230c51669d1fcd35860131e438e234726b2bd5f9adbbd91bd88a718e7e98ecb -genproof
```

### Remarks

* Both `-id` and `-commitmentAtxId` are needed to generate the PoST data.
* If `-id` isn't provided a new identity will be auto-generated. Its private key will be stored in `key.bin` in `-datadir`
with the PoST data. This file then **must** to be copied/moved with the PoST data to run a node with this generated identity.
**NOTE:** The generated PoST data is ONLY valid for this identity!
If a public key is provided with the `-id` flag, the `key.bin` file will be NOT created. Make sure that the key file that belongs
to the identity provided to `postcli` is available in the PoST directory **before** running a node with it.
* `-commitmentAtxId`: it is recommended to look up the highest ATX by querying it from a synced node with
`grpcurl -plaintext -d '' 0.0.0.0:9093 spacemesh.v1.ActivationService.Highest | jq -r '.atx.id.id' | base64 -d | xxd -p -c 64`.
The node can be operated in "non-smeshing" mode during synchronization and when querying the highest ATX.
* The `-reset` flag can be used to clean up a previous initialization. **Careful**: This will delete data that won't be recoverable.

## Initializing a subset of PoST data

It is possible to initialize only subset of the files. This feature is intended to be used to split initialization between many machines.

### Example - split initialization between 2 machines

For this example we initialize 100 units and split the process of initialization into two chunks. This command shows the number of files
that would be created during initialization:

```bash
./postcli -numUnits 100 -printNumFiles
1600
```

**Note:** Ensure that `-id` and `-commitmentAtxId` are the same for all subsets!

On machine A:

```bash
./postcli -numUnits 100 -id <id> -commitmentAtxId <id> -toFile 799 -datadir ./dataA
```

This will create the first 800 files in `./dataA` directory. To verify:

```bash
ls -la ./dataA/*.bin | wc -l
800
```

On machine B:

```bash
./postcli -numUnits 100 -id <id> -commitmentAtxId <id> -fromFile 800 -datadir ./dataB
```

This will create the second 800 files in `./dataB` directory. To verify:

```bash
ls -la ./dataB/*.bin | wc -l
800
```

Finally these sets can be combined by moving/copying the `*.bin` files into the same directory:

```bash
cp ./dataA/*.bin ./data/
cp ./dataB/*.bin ./data/
ls -la ./data/*.bin | wc -l
1600
```

**Merging postmeta_data.json**: Every subset will create its own `postmeta_data.json`. These files MUST be merged manually.
During initialization a VRF nonce is searched that represents the index of the **the label with the smallest value**.

When `postcli` initializes in chunks, each subset can find a valid VRF nonce, which represents the local minimum in the
initialized subset. It is **necessary** to manually select the best one (the global minimum) when merging the subsets.

The VRF nonces are stored as Nonce (index) and NonceValue (the label - 16 byte big endian numbers encoded in hex)
in each `post_metadata.json` of every subset. Given two files:

```json
... (other fields omitted for brevity)
"Nonce": 12345
"NonceValue": "0000ffda94993723a980bf557509773e"
```

```json
... (other fields omitted for brevity)
"Nonce": 98765
"NonceValue": "0000488e171389cce69344d68b66f6b4"
```

The nonce in the second file (please see the `NonceValue` not `Nonce` field) is the global minimum since its value is smaller than the first one. The operator is **required** to find the
smallest VRF nonce by hand and ensure that its index and value are in the `postdata_metadata.json` of the merged directory on the target machine.

Not every chunk will contain a VRF nonce in its `postdata_metadata.json`, but at least one should. If for the very unlikely case that no VRF nonce
was found in any chunk the operator can run `postcli` again **after merging the data** without `-fromFile` and `-toFile` flags to find a VRF nonce.

## Verifying initialized POS data

The `postcli` allows verifying an already initialized POS data. Verification samples a small fraction of labels from every file and compares them to labels generated with the same algorithm executed on CPU. Please note that generating labels on CPU is slow compared to GPU. Hence it is not possible to verify all the data (it would essentially mean re-initialization on CPU). If the GPU failed during initialization, the created PoST data will contain some or all invalid labels after that point. This method will only sample the PoST and might not detect a small amount of corrupted data.

Depending on PoST size and CPU speed a reasonable *fraction* (%) parameter needs to be picked that gives enough confidence but still completes verification in a reasonable time. Suggested values are <1%, closer to 0.1%.

To verify POS data:

1. locate the directory of the POS data. It should contain postdata_metadata.json and postdata_N.bin files.
2. run `postcli -verify -datadir <path to POS directory> -fraction <% of data to verify>`.

For example, `postcli -verify -datadir ~/post/data -fraction 0.1` will verify 0.1% of data. No additional arguments (i.e `-id`) are required. The postcli will read all required information from postdata_metadata.json

If the POS data is found to be invalid, `postcli` will exit with status 1 and print the index of file and offset of the label found to be invalid. If verification completes successfully, `postcli` exits with 0.

## Troubleshooting

### Searching for a lost VRF nonce

In case you lost a VRF nonce after merging initialized subsets, you can use postcli to recover it without re-initializing the data. Postcli will need to **read** the entire POS data and find the nonce.

To find a lost nonce:

1. locate the directory of the POS data. It should contain postdata_metadata.json and postdata_N.bin files.
2. run `postcli -searchForNonce -datadir <path to POS directory>`.

The postcli will read the metadata from postdata_metadata.json and then look for the nonce in all postdata_N.bin files one by one. If the nonce is found it will update the metadata file.
