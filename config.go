package main

import "time"

const cfgInterfacepolling = time.Second * 2
const cfgNetworkscanning = time.Second * 10
const cfgSSHRetry = time.Second * 300
const cfgUpdateIntervalStatic = time.Minute * 15
const cfgUpdateIntervalDynamicRare = time.Minute * 2
const cfgUpdateIntervalDynamicOften = time.Second * 30
