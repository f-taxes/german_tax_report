# German Tax Report

F-Taxes plugin to generate German tax reports.

## FIFO Logic



## German Tax Calculation

At time of buy
Buy 10 BTC @ 300€ + 10€ fee

At time of sell
1 BTC = 400€
1 ETH = 40€

Sell 2 BTC and get 40 ETH.
Total value of bought ETH is 40 * 20€ = 800€
This means the value of the 2 BTC used to buy the ETH is worth 800€
The value of BTC is determined based on the € value of ETH not BTC!
If a fee occurs in BTC we would use the BTC price at time of sell to determine the € value of that fee.
So one might need the ETH €-value AND the BTC €-value at time of transaction.

Need to convert:
- Price of asset to EUR
- Price of quote to EUR

Regarding Fees:
The column "fee" is equal to the "amount" field but for the fee payed.
The column "feePriceC" is equal to "priceC".
This allows to calculate "feeC" which is value the fee payed converted to the base currency.