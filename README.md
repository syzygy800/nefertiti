# Nefertiti
Nefertiti is a FREE crypto trading bot that follows a simple but proven trading strategy; buy the dip and then sell those trades as soon as possible.

### Setup ###

You will need Go installed and `GOPATH` configured.

```bash
mkdir -p $GOPATH/src/github.com/svanas
cd $GOPATH/src/github.com/svanas
git clone https://github.com/svanas/nefertiti.git
```

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
