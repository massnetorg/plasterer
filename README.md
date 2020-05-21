# plasterer

`plasterer` is a fast GPU plotter for [MASS](https://massnet.org/en/).

## Goal

`plasterer` provides a way to plot spaces that are compatible with `massminerd`.

`plasterer-help` cooperates with `plasterer`, it automatically generates config files for `plasterer` and initializes `massminerd`.

`massminerd` is the MassNet full-node miner, see [MassNet-miner](https://github.com/massnetorg/MassNet-miner).

All you will need to do is as simple as:

1. Run `plasterer-help`: generate config files for `plasterer` and initialize `massminerd`.

2. Run `plasterer`: plot all preset spaces.

3. Run `massminerd`: enjoy your solo mining!

**Attention:** `plasterer` only plots 32 GiB (BitLength = 32) massdb files.

## Requirements

Here lists the hardware and software requirements to run `plasterer`:

| Item | Requirement |
| ---- | ----------- |
| OS | Ubuntu 16.04 or newer |
| CPU | 6 cores and 12 threads |
| GPU | GeForce GTX 1060 6 GB |
| Memory | at least 64 GB |
| System Disk | at least 64 GB |
| Data Disk | at least 1 TB |
| CUDA | matched with GPU |

## Usage

### Preparation Stage

Build or Download `plasterer`, `plasterer-helper` and `massminerd`. Put them into a directory like `~/massminer/`.

- `plasterer`, `plasterer-helper`: [Download Page](https://github.com/massnetorg/plasterer/releases)
- `massminerd`: [Download Page](https://github.com/massnetorg/MassNet-miner/releases)

Follow instructions on [MASS Docs](https://docs.massnet.org/en/getting-started/how-to-create-a-full-node-miner/) to create a config file (e.g. `~/massminer/config.json`) for `massminerd`.

Or you can use this sample:

```json
{
  "app": {
    "pub_password": "yourPubPassword"
  },
  "network": {
    "p2p": {
      "seeds": "47.245.28.97,47.254.23.183,47.252.81.90,8.208.26.82,47.56.165.62,106.15.233.21,47.102.141.84,47.104.187.211,47.104.165.118,39.97.225.109,39.97.190.57,118.31.108.197,47.111.120.59,47.111.164.103,112.74.183.26,39.108.215.150,119.23.233.40,47.108.88.140,47.108.89.132,47.108.80.3",
      "listen_address": "tcp://0.0.0.0:43453"
    },
    "api": {
      "api_port_grpc": "9685",
      "api_port_http": "9686"
    }
  },
  "log": {
    "log_level": "info",
    "disable_cprint": false
  },
  "miner": {
    "spacekeeper_backend": "spacekeeper.plasterer",
    "mining_addr": [],
    "generate": false,
    "private_password": "yourPrivatePassword"
  }
}
```

Up to now, you should have `plasterer`, `plasterer-helper`, `massminerd` and `config.json` in directory `~/massminer/`.

### Initialization Stage

Run `plasterer-helper` as:

```
./plasterer-helper init --miner_config <miner_config_file> --miner_priv_pass <miner_private_password> --db_dirs <dir1,dir2,dirN> --db_numbers <num1,num2,numN>
```

The detailed specification for `init` flags is:

```
Usage of init:
  -db_dirs string
        directories for massdb files, separated by comma
  -db_numbers string
        number of massdb files for each directory, separated by comma
  -miner_config string
        miner config file (default "config.json")
  -miner_priv_pass string
        miner private password
```

As `plasterer` only plots 32 GiB (BitLength = 32) massdb files, so you may need to estimate the number of massdb files for each directory.

Or if you leave `-db_numbers` empty, `plasterer-helper` would automatically calculate the proper db_numbers.

After you successfully called `plaster-helper init`, it prints advice to modify `miner_config_file`, please follow it.

**Attention:** `plasterer-help` ignores a directory if the free disk space is less than 128 GiB. Because `plasterer-helper` doesn't make use of all disk space, it always leaves some spare space (`3 * 32 GiB = 96 GiB`) for each directory.

**Tip:** you can also run `./plasterer-help doctor --miner_config <miner_config_file> --db_dirs <dir1,dir2,dirN>` to check the status of your config file and db_dirs.

#### Example

```
./plasterer-helper doctor --miner_config config.json --db_dirs /root/db_dir1/,/root/db_dir2/

Running plasterer-helper doctor...

db_dir: /root/db_dir1
available disk size: 204 GiB
max db number: 3

db_dir: /root/db_dir2
available disk size: 268 GiB
max db number: 5

This is the end of doctor report.
```

```
./plasterer-helper init --miner_config config.json --miner_priv_pass 12345678 --db_dirs /root/db_dir1/,/root/db_dir2/ --db_numbers 2,4

initializing poc wallet: miner
poc wallet initialized: ac10vkeh2xgk9g4lws8v2grszlskxg4pw4xzvu0dmy

Successfully initialized massminerd by plasterer-helper!

Summary for generation:
"directory": /root/db_dir1, "number": 2
"directory": /root/db_dir2, "number": 4

Please manually modify the following items in your miner config file (config.json):
{
  "miner": {
    "spacekeeper_backend": "spacekeeper.plasterer",
    "proof_dir": [
      "/root/db_dir1",
      "/root/db_dir2"
    ],
    "private_password": "12345678"
  }
}

Attention: DO NOT run massminerd while plasterer is running, or massdb files may be corrupted.
```

### Plotting Stage

Once you have finished initialization, Run `plasterer` to plot:

```
./plasterer -q /path/to/db_dir/ -a
```

Usually, `nohup` is recommended to use:

```
nohup ./plasterer -q /path/to/db_dir/ -a > nohup.out 2>&1 &
```

You should run `plasterer` for each directory to finish plotting stage.

Tips: `plasterer` would generate a `core` file if any error occurred, you can manually remove it.

#### Example

```
./plasterer -q /root/db_dir1/ -a

...

b cost time:144009926 us, a cost time:93138431 us

...

b cost time:141718666 us, a cost time:88635845 us
create success, cost time:484590457 us
```

### Mining Stage

**Attention:** DO NOT run `massminerd` while `plasterer` is running, or massdb files could be corrupted.

Once you have finished plotting, Run `massminerd` to enjoy your solo mining:

```
./massminerd
```

## License

`plasterer-helper` is licensed under the terms of the MIT license. See LICENSE for more information or see https://opensource.org/licenses/MIT.
