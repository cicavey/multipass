# Multipass

An auth_request server for nginx which expects to validate a DN post client ssl verification. `Multipass` assumes that the nginx has validated the CA and does not check the certificate itself. 

This server handles the case where simply validating by a CA in not sufficiently narrow - too many users are valid to the CA but only a subset are allowed and we do not control the CA or cert issue process. DNs can spefically be matched using `multipass`.

## Usage

Multipass listens on :4444 and is designed to be used in a container so the port can be mapped. Configuration is loaded from `/etc/multipass/multipass.conf`. This file is a simple line based list of all valid DNs. This list is a whitelist.

Sending a SIGHUP to the process/container will cause `multipass` to reload the configuration and replace existing configuration. SIGTERM or SIGINT will cause the process to exit.

Nginx config block, to included at `server` block level

    # adjust paths for your client CAs
    ssl_client_certificate /etc/nginx/cssl/combined.pem;
    ssl_verify_client optional;

    # Seem CAs require a depth beyond default
    #ssl_verify_depth 10;

    # All other requests will funnel through multipass
    auth_request /auth;

    location = /auth {
      internal;

      # also needed at root nginx level conf if using caching (you should)
      #proxy_cache_path /tmp/nginx-proxy-cache keys_zone=auth_cache:1m;
      proxy_cache auth_cache;
      proxy_cache_valid 5m;
      proxy_cache_key $ssl_client_s_dn;

      # EDIT to where multipass is running
      proxy_pass http://multipass:4444;

      # Needed to pass DN to multipass
      proxy_set_header X-Dn $ssl_client_s_dn;
      proxy_set_header X-Verify $ssl_client_verify;

      # Optimizations
      proxy_pass_request_body off;
      proxy_set_header Content-Length "";
      proxy_set_header X-Original-URI $request_uri;
      proxy_set_header X-Real-Ip $remote_addr;
      proxy_set_header X-Host $host;
      proxy_pass_request_headers on;
    }