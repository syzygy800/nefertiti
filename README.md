### Nefertiti ###

Nefertiti is a FREE crypto trading bot that follows a simple but proven trading strategy; buy the dip and then sell those trades as soon as possible.

### Exchanges ###

At the time of this writing, the trading bot supports the following crypto exchanges: Binance, Kucoin, Bittrex, HitBTC, Coinbase Pro, Bitstamp, CEX.IO, and crypto.com

### Setup ###

You will need Go installed and `GOPATH` configured.

```bash
mkdir -p $GOPATH/src/github.com/svanas
cd $GOPATH/src/github.com/svanas
git clone https://github.com/svanas/nefertiti.git
```

Your developer is using Go version 1.12.7 -- the last version begore Go modules became a thing. This code may or may not compile with other versions of the Go compiler.

### Running ###

```bash
cd $GOPATH/src/github.com/svanas/nefertiti
go build
./nefertiti --help
```

### Testing ###

1. `cd $GOPATH/src/github.com/svanas/nefertiti`
2. `code .`
3. Open the Command Palette (F1)
4. Enter `Go: Test`
5. Click on `Go: Test All Packages in Workspace`
