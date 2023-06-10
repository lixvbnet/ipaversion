# ipaversion

MITM utility for querying iOS App history versions.


![ipaversion_001](_image/ipaversion_01.jpg)

Download from [Releases](./releases) page.



## prerequisites

1. Install [iTunes 12.6.5.3](https://secure-appldnld.apple.com/itunes12/091-87819-20180912-69177170-B085-11E8-B6AB-C1D03409AD2A6/iTunes64Setup.exe) , login Apple ID, and trust & authorize the computer you are using.
2. Download `ipaversion` from [Releases](./releases) page.
3. Generate your own CA cert and put into `~/.mitmproxy` , **OR** just start `ipaversion` and it will generate a new one.
4. Trust the CA cert. (Read mitmproxy docs [About Certificates](https://docs.mitmproxy.org/stable/concepts-certificates/) for more information)

- macOS

```shell
sudo security add-trusted-cert -d -p ssl -p basic -k /Library/Keychains/System.keychain ~/.mitmproxy/mitmproxy-ca-cert.pem
```

- Windows

```shell
certutil -addstore root mitmproxy-ca-cert.cer
```

Now you are ready to GO!



## usage

1. Quit `ipaversion` if it is open. Run `ipaversion -c` to make sure it turns off the system proxy.
2. Open iTunes, search for an App. Buy it if you haven't.
3. Start `ipaversion` 
4. Go back to iTunes and click Download button.
5. `ipaversion` will intercept the request and use it to get history versions.



## help

```shell
$ ipaversion -h
Usage: ipaversion [options]
options
  -c    cleanup and exit. (e.g. turn off proxy)
  -end int
        versionIDs index range [start, end) (default 9223372036854775807)
  -h    show help and exit
  -ps
        show current system proxy status
  -s    do not set system proxy
  -start int
        versionIDs index range [start, end)
  -v    show version
```



## features

- [x] Set system proxy when starting `ipaversion` , and turn off proxy upon normal quitting.
- [x] Add `-c` option for cleanup. (e.g. turn off system proxy)
- [x] Add `-s` option: do not set system proxy.
- [x] Query a given range of the history versions.
- [ ] Enter index number to download the `ipa` (TODO: still working on generating `iTunesMetadata.plist`).

