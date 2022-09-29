# Uptime Checker Actor

# Introduction
This repo acts as a FVM registry for node uptime checking. At a basic level, many use cases require a way of knowing 
which nodes are active/live in a p2p network. The purpose is to have a set of, what we call, checkers constantly ping 
those nodes to see which ones are still active and which ones are down. 

At the same time, the checkers will crosscheck each other to ensure checkers are also online. If any checker is found 
to be done, voting would be carried out among checkers. If a quorum of the voters has reported a checker to be down,
then that checker will be removed from the registry

This project contains two parts:
* FVM actor (this repo)
* [Go Checker](https://github.com/cryptoAtwill/uptime-checker-golang)

This `FVM actor` is the registry running in a FVM. It provides the CRUD functions for nodes and checkers registration 
lifecycle. The `Go Checker` is the checker implementation in golang that ping the nodes/checkers, provide node uptime 
informations, see more details in the repo linked above.

## Build
To compile:
```shell
cargo build
```
You should be able to see the `uptime_checker.compact.wasm` compiled generated.

Set up a local fvm according to this [tutorial](https://lotus.filecoin.io/lotus/developers/local-network/).

Deploy the actor:
```shell
./lotus chain install-actor uptime_checker.compact.wasm
```
Once the transaction is processed, obtain the message cid.

Create the actor with
```shell
./lotus chain create-actor ${MESSAGE_CID} ewogICAgImlkcyI6IFtdLAogICAgImNyZWF0b3JzIjogW10sCiAgICAiYWRkcmVzc2VzIjogW10KfQ==
```
In the above command, `ewogICAgImlkcyI6IFtdLAogICAgImNyZWF0b3JzIjogW10sCiAgICAiYWRkcmVzc2VzIjogW10KfQ` is the base64 encoded 
json string. You can replace with other configurations.

Once you obtain the address, you can interact with the actor. Use the following template to operate:
```shell
./lotus chain invoke <METHOD_NUMBER> <PAYLOAD>
```
Refer to `UptimeCheckerActor` trait in `src/traits.rs` for more reference. 
