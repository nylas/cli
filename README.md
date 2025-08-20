# Nylas CLI
The Nylas CLI is an intermediary between Nylas servers and the local machine, bridging the webhook connection and creating a tunnel which can be used to test various aspects of the webhook client.

# Documentation
...

# Installation
To install the Nylas CLI, use `brew install nylas-cli` or `npm install nylas-cli`.

# Alternatives
An alternative to this CLI is to use [cloudflared](https://github.com/cloudflare/cloudflared), which can be used to create a temporary tunnel pointing to a randomized public URL. Using this URL as the webhook's URL in the Nylas Dashboard will replicate  the functionality of this CLI, though the webhook URL will have to be updated each time the cloudflared tunnel is restarted, as the generated public URLs are not consistent.