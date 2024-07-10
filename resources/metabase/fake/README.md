# Metabase
Metabase expects an PKCS#8 private key, so we generate a fake one with the following command:

```bash
openssl genpkey -out rsakey.pem -algorithm RSA -pkeyopt rsa_keygen_bits:2048
```

Then we copy it into the `fake-metabase-sa.json` file.
