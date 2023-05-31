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

https://amdgpu-install.readthedocs.io/en/latest/install-script.html

### Intel

https://github.com/intel/compute-runtime/releases

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
./postcli -provider=2 -id=c230c51669d1fcd35860131e438e234726b2bd5f9adbbd91bd88a718e7e98ecb -commitmentAtxId=c230c51669d1fcd35860131e438e234726b2bd5f9adbbd91bd88a718e7e98ecb -genproof
 
```

### Remarks
* `-id` and `-commitmentAtxId` are required because they are committed to the generated data.
* If `-id` isn't provided, the id (public key) will be auto-generated, while saving `key.bin` in `-datadir`.
* If `postcli` is called multiple times on a given `-datadir`, config mismatch error is likely to occur. In this case, the `-reset` flag can be used to easily clean the previous instance. 
