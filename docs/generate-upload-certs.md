# Generating and Upload Certs

*This process needs to be automated!*

* Generate Let's Encrypt certs for [atlantis.run](https://atlantis.run)

```bash
sudo certbot certonly --manual --server https://acme-v01.api.letsencrypt.org/directory -d atlantis.run -d www.atlantis.run
```

Follow the instructions after running the command

* Upload the certs to AWS to be used by Cloudfront

```bash
sudo aws iam upload-server-certificate --server-certificate-name atlantis_run_lets_encrypt_cert --certificate-body file:///etc/letsencrypt/live/atlantis.run/cert.pem --private-key file:///etc/letsencrypt/live/atlantis.run/privkey.pem --certificate-chain file:///etc/letsencrypt/live/atlantis.run/chain.pem --path /cloudfront/certs/
```