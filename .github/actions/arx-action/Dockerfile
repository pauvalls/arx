# This Dockerfile is kept for reference but the action now uses
# a composite approach (action.yml runs: using: "composite").
# The composite action builds arx from source at runtime,
# avoiding Docker build context restrictions for local actions.
#
# To use as a Docker action instead (requires binary in this directory):
#   1. Build arx and copy to .github/actions/arx-action/arx
#   2. Change runs.using in action.yml to "docker"
#   3. Enable the lines below and disable the composite step.
#
# FROM alpine:3.20
# RUN apk add --no-cache ca-certificates bash
# COPY arx /usr/local/bin/arx
# COPY entrypoint.sh /entrypoint.sh
# RUN chmod +x /entrypoint.sh
# ENTRYPOINT ["/entrypoint.sh"]
