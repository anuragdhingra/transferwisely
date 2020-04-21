# transferwise-go
Batch process using transfer-wise API to automatically detect better rates and book transfers for you.

### Why do we need this?
[Transferwise](https://transferwise.com/) is a leading online money transfer service. 
One can book a quote at live mid-market rate that further can be used to create a transfer. 
This transfer is a payment order to recipient account based on a quote. Once created, a transfer needs to be funded within the next five working days. Otherwise, it will be automatically canceled.
Its very easy to miss the best rates and kinda difficult to keep checking for them all the time if you are planning to make a transfer or if you do it too often, 
thus this was my naive attempt at solving this problem.


### Steps to follow
- Pull the pre-built image from docker hub using:

```bash
docker pull anuragdhingra/transferwise-batch:v1.0
```

- Run the image passing the following env variables:

```bash
docker run \
--name transferwise-batch-sandbox \
-d \
-e ENV=production \
-e API_TOKEN=<YOUR API TOKEN> \
-e MARGIN=0.001 \
-e INTERVAL=1 \
anuragdhingra/transferwise-batch:v1.0
```

`ENV`: `sandbox` or `production`

`API_TOKEN`: Generate it from [here](https://transferwise.com/help/19/transferwise-for-business/2958229/whats-a-personal-api-token-and-how-do-i-get-one).
_Note: Sandbox and production environment have different API Tokens._

`MARGIN` (optional, defaults to 0): Currency margin at which you want to book a new transfer, value defaults to 0 i.e any higher rate
When `MARGIN`=0.01
_(Please set it according to your respective currency rate change in absolute terms)_
For example -
```
Current booked transfer : JPY --> INR : Rate --> 0.691
Live Rate --> 0.695 : NO ACTION NEEDED
Live Rate --> 0.711 : NEW TRANSFER BOOKED, cancelling the old one
```

`INTERVAL` (optional, defaults to 1): Time(in minutes) interval at which you want to query transferwise to check for better rates

### Other things to note before using this on production:
- Currently, it doesnt supports creating a quote/transfer if there is no existing transfer at the moment. 
The reason to this being all the info regarding the new transfer to be made like recipient account,amount etc. is taken from the existing transfer.
- At the moment, as transferwise at maximum blocks live rate for first three of all your transfers booked.
Thus, the batch also gets only the first three or less existing transfers you have to compare for better rates.


### Improvements

- Support creating a new transfer if no current transfer is present.
- Alert the user via email/SMS if a new transfer is booked or if a best rate transfer is about to expire.
- Add testing


### Contribute
Open up an issue or send in a pull request if you'd like to contribute.







