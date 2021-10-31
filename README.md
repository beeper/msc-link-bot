# msc-link-bot

This is re-write of `@msclinkbot:matrix.org` in golang with support for
encrypted rooms with the help of <https://maunium.net/go/mautrix>.

# Usage

```sh
# will create ./msc-link-bot
make

export HOMESERVER=https://matrix.example.org
export USER_ID=@msclinkbot:example.org
export DEVICE_ID=FWQXHAAVLA
export ACCESS_TOKEN=<super_secret_access_token>

# crypto keys will be stored in ./crypto.db
./msc-link-bot
```

Those looking to use docker should see [contrib/README.md](contrib/README.md)
