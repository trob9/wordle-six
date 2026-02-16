#!/bin/sh
# Fix volume permissions for non-root user.
# Docker volumes created by root-user containers retain root ownership,
# so we chown at startup before dropping to appuser.
chown -R appuser:appuser /data
exec su-exec appuser ./wordle-six
