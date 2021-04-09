# Nefertiti

Nefertiti is a FREE crypto trading bot that follows a simple but proven trading strategy; buy the dip and then sell those trades as soon as possible.

## Exchanges

At the time of this writing, the trading bot supports the following crypto exchanges:

<a href="https://www.bitstamp.net/ref/QWE1MDzZoyPWZNyU/"><img src="https://nefertiti-tradebot.com/wp-content/uploads/2019/12/bitstamp-logo.png"></a> &nbsp;
<a href="https://bittrex.com/Account/Register?referralCode=CIC-YDN-5DX"><img src="https://nefertiti-tradebot.com/wp-content/uploads/2019/12/bittrex_logo-1.png"></a> &nbsp;
<a href="https://hitbtc.com/?ref_id=5aad6226b7072"><img src="hhttps://nefertiti-tradebot.com/wp-content/uploads/2019/12/hitbtc-logo.png"></a> &nbsp;
<a href="https://pro.coinbase.com/"><img src="https://nefertiti-tradebot.com/wp-content/uploads/2019/12/gdax_logo.png"></a> &nbsp; &nbsp; &nbsp;
<a href="https://www.binance.com/en/register?ref=UME24R7B"><img src="https://nefertiti-tradebot.com/wp-content/uploads/2019/12/binance_logo.png"></a> &nbsp; &nbsp; &nbsp;
<a href="https://www.kucoin.com/?rcode=KJ6stw"><img src="https://nefertiti-tradebot.com/wp-content/uploads/2019/12/KuCoin-logo-1.png"></a> &nbsp; &nbsp; &nbsp;
<a href="https://crypto.com/exch/rf3v8ucd4k"><img src="https://nefertiti-tradebot.com/wp-content/uploads/2020/09/crypto-com-review.png"></a>

## Signals

At the time of this writing, the trading bot supports the following signal providers:

<a href="https://www.mininghamster.com/referral/azr8N29xml4dq4GpbzxTNuB3DZpfCxzA"><img src="https://nefertiti-tradebot.com/wp-content/uploads/2018/04/mininghamster.jpg" width="100"></a> &nbsp;  &nbsp;  &nbsp;
<a href="https://premium.cryptoqualitysignals.com/register/WYn"><img src="https://nefertiti-tradebot.com/wp-content/uploads/2019/01/1_Sa5hV8OSo2Kgsv7hq3OACw.jpeg" width="100"></a> &nbsp;  &nbsp;  &nbsp;
<a href="https://altrady.com/?a=nefertiti"><img src="https://nefertiti-tradebot.com/wp-content/uploads/2019/02/icon-1024x1024-300x300.png" width="100"></a>

## Setup

Download and install Go quickly with the steps described here.

### Go download
Click the button below to download the Go installer.

<a href="https://golang.org/dl/"><img src="https://i.ibb.co/gJyVCcJ/pngegg.png" width="120"></a>

### Go install
#### Linux
1. Extract the archive you downloaded into /usr/local, creating a Go tree in /usr/local/go.

    <b>Important:</b> This step will remove a previous installation at /usr/local/go, if any, prior to extracting. Please back up any data before proceeding.

    For example, run the following as root or through sudo:

    ```
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.16.3.linux-amd64.tar.gz
    ```

2. Add /usr/local/go/bin to the PATH environment variable.
   You can do this by adding the following line to your $HOME/.profile or /etc/profile (for a system-wide installation):
   
    ```
    export PATH=$PATH:/usr/local/go/bin
    ```

    <b>Note:</b> Changes made to a profile file may not apply until the next time you log into your computer. To apply the changes immediately, just run the shell commands directly or execute them from the profile using a command such as source $HOME/.profile.

3. Verify that you've installed Go by opening a command prompt and typing the following command:

    ```
    $ go version
    ```

4. Confirm that the command prints the installed version of Go.

#### Mac
1. Open the package file you downloaded and follow the prompts to install Go.
   
    The package installs the Go distribution to /usr/local/go. The package should put the /usr/local/go/bin directory in your PATH environment variable. You may need to restart any open Terminal sessions for the change to take effect.
    
2. Verify that you've installed Go by opening a command prompt and typing the following command:

    ```
    $ go version
    ```

3. Confirm that the command prints the installed version of Go.

#### Windows
1. Open the MSI file you downloaded and follow the prompts to install Go.

    By default, the installer will install Go to Program Files or Program Files (x86). You can change the location as needed. After installing, you will need to close and reopen any open command prompts so that changes to the environment made by the installer are reflected at the command prompt.
 
2. Verify that you've installed Go.

      1. In <b>Windows</b>, click the <b>Start</b> menu.
      2. In the menu's search box, type cmd, then press the <b>Enter</b> key.
      3. In the Command Prompt window that appears, type the following command:
  
     <br>
  
      ```
      $ go version
      ```
      
     4. Confirm that the command prints the installed version of Go.

## Cloning
You will need Go installed and `$GOPATH` configured.

  ```
  mkdir -p $GOPATH/src/github.com/svanas
  cd $GOPATH/src/github.com/svanas
  git clone https://github.com/svanas/nefertiti.git
  ```

## Updating
Update your local working branch with commits from the remote, and update all remote tracking branches.

  ```
  cd $GOPATH/src/github.com/svanas
  git pull
  ```

## Notes
Your developer is using the latest Go version (https://golang.org/dl/) -- This code may or may not compile with other versions of the Go compiler.

## Dependencies

Most dependencies are vendored in with this repo. You might need to clone the following repositories:
* https://github.com/svanas/go-binance
* https://github.com/svanas/go-crypto-dot-com
* https://github.com/svanas/go-mining-hamster

<br>

  ```
  cd $GOPATH/src/github.com/svanas
  git clone https://github.com/svanas/go-binance.git
  git clone https://github.com/svanas/go-crypto-dot-com.git
  git clone https://github.com/svanas/go-mining-hamster.git
  ```

## Running

```
cd $GOPATH/src/github.com/svanas/nefertiti
go build
./nefertiti --help
```
