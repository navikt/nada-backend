# dp

An application to register and publish dataproducts, as well as to give access to dataproducts.

⚠️ Currently in progress, expect an MVP soon™

- [ ] Closed beta testing
- [ ] General availability

Read more about specifics of `dp` in our [docs](./docs/README.md)

## What?

Let's say you own some data in a database some place. 
You want to make the data available for someone without giving them direct access to the database.
Instead, you agree to periodically publish the set of data to a [Bucket](https://cloud.google.com/storage/docs/key-terms#buckets) or to [BigQuery](https://cloud.google.com/bigquery).
Then, you give access to subject who wants to read this data.

This is what `dp` aims to help streamline.

- Provision a Bucket or BigQuery in your `nais.yaml` as the datastore
- Set up a [job](https://doc.nais.io/addons/naisjobs/) to periodically publish data to the datastore, and update the dataproduct entry in `dp` (so we can keep track)
- Let the consumers of your data gain access by themselves through `dp`
