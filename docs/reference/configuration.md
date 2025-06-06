# Configuration reference

This page references the different configurations for the Scaleway fleeting plugin.

[TOC]

## Plugin configuration

The [`[runners.autoscaler.plugin_config]` section](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnersautoscalerplugin_config-section) supports the following parameters:

<table>
  <tr>
    <th>Parameter</th>
    <th>Type</th>
    <th>Description</th>
  </tr>
  <tr>
    <td><code>name</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      Name of the fleeting plugin instance group. The created instance names will be
      prefixed using this name.
    </td>
  </tr>
  <tr>
    <td><code>access_key</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      <a href="https://www.scaleway.com/en/docs/iam/how-to/create-api-keys/">Scaleway API access key</a>
      to access your Scaleway project.
      <br>
      You may also use the <code>SCW_ACCESS_KEY</code> environment variable to configure the value.
    </td>
  </tr>
  <tr>
    <td><code>secret_key</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      <a href="https://www.scaleway.com/en/docs/iam/how-to/create-api-keys/">Scaleway API secret key</a>
      to access your Scaleway project.
      <br>
      You may also use the <code>SCW_SECRET_KEY</code> environment variable to configure the value.
    </td>
  </tr>
  <tr>
    <td><code>organization</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      <a href="https://www.scaleway.com/en/docs/organizations-and-projects/concepts/">Scaleway Organization ID</a>
      to connect to.
      <br>
      You may also use the <code>SCW_ORGANIZATION_ID</code> environment variable to configure the value.
    </td>
  </tr>
  <tr>
    <td><code>project</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      <a href="https://www.scaleway.com/en/docs/organizations-and-projects/concepts/">Scaleway Project ID</a>
      to connect to.
      <br>
      You may also use the <code>SCW_PROJECT_ID</code> environment variable to configure the value.
    </td>
  </tr>
  <tr>
    <td><code>endpoint</code></td>
    <td>string</td>
    <td>
      Scaleway API endpoint to use.
      <br>
      You may also use the <code>SCW_API_URL</code> environment variable to configure the value.
    </td>
  </tr>
  <tr>
    <td><code>zone</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      <a href="https://www.scaleway.com/en/docs/account/reference-content/products-availability/">Scaleway Zone</a>
      to deploy to.
      <br>
      You may also use the <code>SCW_DEFAULT_ZONE</code> environment variable to configure the value.
    </td>
  </tr>
  <tr>
    <td><code>server_type</code></td>
    <td>string or list of string (<strong>required</strong>)</td>
    <td>
      <a href="https://www.scaleway.com/en/docs/account/reference-content/products-availability/">Scaleway server type</a>
      on which the instances will run. Using a list of server types allows you to define
      additional server types to fallback to in case of unavailable resource errors. All
      servers types must have the same CPU architecture.
      <br>
      You can list the available server types by running <code>scw instance server-type list</code>.
    </td>
  </tr>
  <tr>
    <td><code>image</code></td>
    <td>string (<strong>required</strong>)</td>
    <td>
      Scaleway image from which the instances will run. It supports both marketplace and private images.
      <br>
      You can list the available images by running <code>scw marketplace image list</code> or <code>scw instance image list</code>.
    </td>
  </tr>
  <tr>
    <td><code>public_ipv4_disabled</code> and <code>public_ipv6_disabled</code></td>
    <td>boolean</td>
    <td>
      Disable the instances public IPv4/IPv6.
    </td>
  </tr>
  <tr>
    <td><code>user_data</code> and <code>user_data_file</code></td>
    <td>string</td>
    <td>
      Configuration for the provisioning utility that runs during the instances creation.
      On Ubuntu, you can provide a Cloud Init configuration to setup the instances. Make
      sure to wait for the instances to be ready before scheduling jobs on them by using
      the autoscaler <code>instance_ready_command</code> config.
      Note that <code>user_data</code> and <code>user_data_file</code> are mutually exclusive.
    </td>
  </tr>
  <tr>
    <td><code>volume_size</code></td>
    <td>integer</td>
    <td>
      Size in GB for the root <a href="https://www.scaleway.com/en/docs/instances/concepts/#block-volumes">Volume</a>
      that will be attached to each instance. The minimal <code>volume_size</code> is 10 GB.
    </td>
  </tr>
</table>

## Autoscaler configuration

Below are parameters from the [`[runners.autoscaler]` section](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnersautoscaler-section) that are important for our plugin:

<table>
  <tr>
    <th>Parameter</th>
    <th>Description</th>
  </tr>
  <tr>
    <td><code>instance_ready_command</code></td>
    <td>
      When using the <code>user_data</code> or <code>user_data_file</code> config, you
      must wait for the instances to be ready before scheduling jobs on them. When using
      Cloud Init, this can be done with the following: <code>cloud-init status --wait || test $? -eq 2</code>
    </td>
  </tr>
</table>

## Connector configuration

Below are parameters from the [`[runners.autoscaler.connector_config]` section](https://docs.gitlab.com/runner/configuration/advanced-configuration.html#the-runnersautoscalerconnector_config-section) that are important for our plugin:

<table>
  <tr>
    <th>Parameter</th>
    <th>Value</th>
  </tr>
  <tr>
    <td><code>use_external_addr</code></td>
    <td>
      Access the instances through their public addresses. Note that without private
      networks, this field must be set to <code>true</code>.
    </td>
  </tr>
  <tr>
    <td><code>os</code></td>
    <td>Only <code>linux</code> is supported.</td>
  </tr>
    <tr>
    <td><code>protocol</code></td>
    <td>Only <code>ssh</code> is supported.</td>
  </tr>
</table>
