# wiki
File based personal wiki.

**Currently under development.**


## Install
- You can download a tarball with a prebuilt executable for your platform [here](https://github.com/BenDoan/wiki/releases)
- Unpack the tarbal with ```tar -xf wiki-x.x.x.tar.gz```
- Start the wiki with ```./wiki```
- (Optional) Edit ```config.toml``` to change the configuration
- (Optional) After an account is registered, it's role will need to be changed to admin to view or edit the wiki. To do this: in data/users.txt change the 3rd field of that user to 1

## Build
- To download dependencies run ```go get```
- To build run ```go build```
