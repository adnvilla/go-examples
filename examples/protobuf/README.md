

Download and install the protocol buffer compiler

https://github.com/protocolbuffers/protobuf/releases

For windows 
https://github.com/protocolbuffers/protobuf/releases/download/v3.6.1/protoc-3.6.1-win32.zip

Unzip and add directory in PATH, example:
C:\protoc-3.6.1-win32\bin

Tutorial
https://developers.google.com/protocol-buffers/docs/gotutorial

protoc -I=./ --go_out=./ ./addressbook.proto