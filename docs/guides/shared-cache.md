# Enable shared cache using the Scaleway Object Storage

A local cache provides limited caching capabilities for your CI jobs running across many instances. This document describes the steps to enable a shared cache using the [Scaleway Object Storage](https://www.scaleway.com/en/docs/object-storage/) and the Scaleway fleeting plugin.

## Creating S3 credentials and bucket

First, you must generate S3 credentials using the ["Create API keys" guide](https://www.scaleway.com/en/docs/iam/how-to/create-api-keys/).

Once you have the credentials, you must create a new S3 Bucket using the ["Create a Bucket" guide](https://www.scaleway.com/en/docs/object-storage/how-to/create-a-bucket/).

> It is recommended to add a random suffix to the name of your Bucket, e.g. `gitlab-ci-cache-7d2f6722`.

## Configuring the `gitlab-runner`

Now that you gathered all the required data, you can configure the `gitlab-runner` to enable the S3 shared cache:

```toml
[runners.cache]
Type = "s3"
Shared = true

[runners.cache.s3]
ServerAddress = "s3.fr-par.scw.cloud"
BucketName = "gitlab-ci-cache-7d2f6722"
AccessKey = "SCWXXXXXXXXXXXXXXXXX"
SecretKey = "b78cf38b-cbf3-47c8-b729-fb1069a9d4a2"
```

> Make sure to update the:
>
> - `ServerAddress` with the location you chose.
> - `BucketName` with the Bucket name you chose.
> - `AccessKey` with the access key you generated.
> - `SecretKey` with the secret key you generated.

For more details about the `[runners.cache]` config, see the [`gitlab-runner` cache configuration reference](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnerscache-section).
