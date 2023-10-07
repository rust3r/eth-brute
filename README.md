# eth-brute

Bruteforce with sequential, random private keys, brainwallets


## Compile from source

```
git clone git@github.com:rust3r/eth-brute.git
cd eth-blute/
go get
go build
```


## Docker

```
docker build -t app .
docker run -it --rm -p 8545:8545 app
```


## Examples

For help:

```
./eth-brute -h

Usage of ./eth-brute:
  -brain string
        Password list
  -pk string
        Start private key
  -port int
        Ethereum rpc port (default 8545)
  -random
        Generate random private key
  -server string
        Ethereum rpc server (default "202.61.239.89")
  -threads int
        Number of threads (default 4)
```

Usage:

```
./eth-brute -threads 50 -pk 1206b75e20883f695a49bbabd1f26e5d30afb75e4a9e3989a89d779d4a3a1c92
./eth-brute -threads 50 -random
./eth-brute -threads 50 -brain passwords_example.txt
```

Spare servers can be found in the source code


## Donation
 Eth: 0x7630b188630841ce9c1e1eDa7a173FC344458f62
