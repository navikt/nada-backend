# dp frontend

Satt opp med [NAVs create-react-app](https://github.com/navikt/nav-frontend-moduler/tree/master/examples/cra), se detaljer der

## Konfigurasjon

Både auth callback URI og API-rot utledes fra miljøvariablen BACKEND_ENDPOINT, med forvalg 'http://localhost:8080'

## Kjøring

Development og linting:

```
yarn install
yarn start
```

Bygging for produksjon:

```
yarn install
yarn run build
```
