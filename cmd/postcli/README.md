# postcli

CLI tool for PoST initialization

# Getting it

Go to the https://github.com/spacemeshos/post/releases and take the most recent release for your platform. In case if you want to build it from source, follow the instructions below.

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


### How to set the required options

* `-commitmentAtxId`: At the moment, there is no easy way to get that programaticaly. If you start to initialize before first 2 epochs finish then it's safe to use `goldenATX` value here. Later the best way to get it is to use grpc api of the node and call `ActivationService::Highest()` to get that value.
* `-id`: There are two options there, you can autogenerate it (do not provide it as an option) then there will be `key.bin` file in the `-datadir` folder which will have private key *REMEMBER* to copy it also. The other option is to provide the `-id` flag with the public key you want to use. In this case, the `key.bin` file will be NOT craeted. Make sure that you put the binary representation of the key there.
