# LedgerStockUpdate

This application locates any stocks/crypto currencies you have in your [ledger-cli](https://ledger-cli.org) file, then generates a price database of those stocks compatible with the application, using quotes from Yahoo Finance.

Forked from [adchari/LedgerStockUpdate](https://github.com/adchari/LedgerStockUpdate), add some features:
* Add name mapping functionality: use BTC in your ledger and BTC-USD for online quotes
* Replace API from Alpha Vantage, without the needs of obtaining API token
* Recursively transform the quotes to single currency

### Usage

Build the go file, and run as follows:

```bash
./[name of executable] -f=[ledger file] -p=[price database file (to create or update)] -b=[Name of ledger binary] -m=[name mapping file]
```

This should spit out a price database file, which can then be used to calculate the market value in ledger as follows:

```bash
ledger -f [ledger file] --price-db [price database file] -V bal
```

