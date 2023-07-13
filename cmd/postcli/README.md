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

###  Print the number of files that would be initialized

```bash
./postcli -numUnits 100 -printNumFiles
1600
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

### Initialize subset
It is possible to initialize only subset of the files. It's useful if one wants to split initialization between many machines.
#### Example - split initialization between 2 machines
Let's assume one wants to initialize 100 units and split the process of initialization to machine A and B.

First let's see how many files it would be:
```bash
./postcli -numUnits 100 -printNumFiles
1600
```

Next, on machine A:
```bash
./postcli -numUnits 100 -id <id> -commitmentAtxId <id> -toFile 799 -datadir ./dataA
```
We get the first half - 800 binary files
```bash
ls -la ./dataA/*.bin | wc -l
800
```

Next, on machine B:
```bash
./postcli -numUnits 100 -id <id> -commitmentAtxId <id> -fromFile 800 -datadir ./dataB
```
We get the second half - 800 binary files
```bash
ls -la ./dataB/*.bin | wc -l
800
```

Finally we can combine both sets together. A dummy example to get the feeling. Realisticly it would probably be copying over the network
```bash
cp ./dataA/* ./data/
cp ./dataB/* ./data/
ls -la ./data/*.bin | wc -l
1600
```

**An optional step to select best possible VRF nonce**
Normally, when `postcli`initializates from the start to the end it will automatically pick the best VRF nonce. The best means pointing to **the label with the smallest value**. This will avoid a longer initialization time when a node increases their PoST size in the future (not supported yet).

Now, when `postcli` initializes in chunks, each subset will find a valid vrf nonce, which represents the local minimum in the inititalized subset. It is recommended to select the best one (the global minimum).

The values of nonces are 128bit, represented as a 16B binary array in big endian. Given two nonces:
```
NonceA = 12345
NonceValueA = 0000ffda94993723a980bf557509773e
NonceB = 98765
NonceValueB = 0000488e171389cce69344d68b66f6b4
```
`NonceB` is the global minimum since its value is smaller than the one of `NonceA`.

The nonce (index) and noncevalue (the label) is included in the post_metadata.json. It is up to the operator to find the best VRF nonce manually and copy the `Nonce` and `NonceValue` to the postdata_metadata.json on the target machine.

### Remarks
* `-id` and `-commitmentAtxId` are required because they are committed to the generated data.
* If `-id` isn't provided, the id (public key) will be auto-generated, while saving `key.bin` in `-datadir`.
* If `postcli` is called multiple times on a given `-datadir`, config mismatch error is likely to occur. In this case, the `-reset` flag can be used to easily clean the previous instance.


### How to set the required options

* `-commitmentAtxId`: At the moment, there is no easy way to get that programaticaly. If you start to initialize before first 2 epochs finish then it's safe to use `goldenATX` value here. Later the best way to get it is to use grpc api of the node and call `ActivationService::Highest()` to get that value.
* `-id`: There are two options there, you can autogenerate it (do not provide it as an option) then there will be `key.bin` file in the `-datadir` folder which will have private key *REMEMBER* to copy it also. The other option is to provide the `-id` flag with the public key you want to use. In this case, the `key.bin` file will be NOT craeted. Make sure that you put the binary representation of the key there.
