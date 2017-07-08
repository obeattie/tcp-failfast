# `tcp-failfast`

[![Build Status](https://travis-ci.org/obeattie/tcp-failfast.svg?branch=master)](https://travis-ci.org/obeattie/tcp-failfast)

`tcp-failfast` is a Go library which allows control over the TCP "user timeout"
behavior.

This timeout is specified in RFC 793 but is not implemented on all platforms. Currently Linux and Darwin are supported.