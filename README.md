# msc-link-bot

This is a re-write of `@msclinkbot:matrix.org` in golang with support for
encrypted rooms with the help of <https://maunium.net/go/mautrix>.

# Usage

```sh
# will create ./msc-link-bot
make

cp example-config.yaml config.yaml
# edit your configuration to fit your needs

# crypto keys will be stored in ./crypto.db
./msc-link-bot
```

Those looking to use docker should see [contrib/README.md](contrib/README.md)
