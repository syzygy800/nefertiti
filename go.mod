module github.com/svanas/nefertiti

go 1.16

require (
	github.com/Kucoin/kucoin-go-sdk v1.2.8
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/alexflint/go-filemutex v1.1.0
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/bitbandi/go-hitbtc v0.0.0-20190201230334-2adae5a2f724
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/dghubble/go-twitter v0.0.0-20201011215211-4b180d0cc78d
	github.com/dghubble/oauth1 v0.7.0
	github.com/equinox-io/equinox v1.2.0
	github.com/fatih/color v1.12.0 // indirect
	github.com/go-errors/errors v1.4.0
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/gregdel/pushover v1.1.0
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/mitchellh/cli v1.1.2
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/posener/complete v1.2.3 // indirect
	github.com/preichenberger/go-coinbase-exchange v0.0.0-20170804193904-a283500f7727
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/svanas/go-binance v0.0.0-20210426082427-201a69b9b187
	github.com/svanas/go-crypto-dot-com v0.0.0-20200621190630-a4cbac992669
	github.com/svanas/go-mining-hamster v0.0.0-20190102110438-73bc620cc6e9
	github.com/yanzay/tbot v1.0.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/sys v0.0.0-20210525143221-35b2ab0089ea // indirect
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56 // indirect
)

replace github.com/Kucoin/kucoin-go-sdk => ./vendor-modified/github.com/Kucoin/kucoin-go-sdk

replace github.com/bitbandi/go-hitbtc => ./vendor-modified/github.com/bitbandi/go-hitbtc

replace github.com/go-errors/errors => ./vendor-modified/github.com/go-errors/errors

replace github.com/go-telegram-bot-api/telegram-bot-api => ./vendor-modified/github.com/go-telegram-bot-api/telegram-bot-api

replace github.com/preichenberger/go-coinbase-exchange => ./vendor-modified/github.com/preichenberger/go-coinbase-exchange
