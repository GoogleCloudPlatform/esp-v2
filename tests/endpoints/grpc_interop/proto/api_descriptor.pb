
�9

test.protogrpc.testing"
Empty"L
Payload-
type (2.grpc.testing.PayloadTypeRtype
body (Rbody"�
SimpleRequest>
response_type (2.grpc.testing.PayloadTypeRresponseType#
response_size (RresponseSize/
payload (2.grpc.testing.PayloadRpayload#
fill_username (RfillUsername(
fill_oauth_scope (RfillOauthScope"~
SimpleResponse/
payload (2.grpc.testing.PayloadRpayload
username (	Rusername
oauth_scope (	R
oauthScope"L
StreamingInputCallRequest/
payload (2.grpc.testing.PayloadRpayload"T
StreamingInputCallResponse6
aggregated_payload_size (RaggregatedPayloadSize"I
ResponseParameters
size (Rsize
interval_us (R
intervalUs"�
StreamingOutputCallRequest>
response_type (2.grpc.testing.PayloadTypeRresponseTypeQ
response_parameters (2 .grpc.testing.ResponseParametersRresponseParameters/
payload (2.grpc.testing.PayloadRpayload"N
StreamingOutputCallResponse/
payload (2.grpc.testing.PayloadRpayload*?
PayloadType
COMPRESSABLE 
UNCOMPRESSABLE

RANDOM2�
TestService5
	EmptyCall.grpc.testing.Empty.grpc.testing.EmptyF
	UnaryCall.grpc.testing.SimpleRequest.grpc.testing.SimpleResponsel
StreamingOutputCall(.grpc.testing.StreamingOutputCallRequest).grpc.testing.StreamingOutputCallResponse0i
StreamingInputCall'.grpc.testing.StreamingInputCallRequest(.grpc.testing.StreamingInputCallResponse(i
FullDuplexCall(.grpc.testing.StreamingOutputCallRequest).grpc.testing.StreamingOutputCallResponse(0i
HalfDuplexCall(.grpc.testing.StreamingOutputCallRequest).grpc.testing.StreamingOutputCallResponse(0J�,
 �
�
 w An integration test service that covers all the method signature permutations
 of unary/streaming requests/responses.
2� Copyright 2017 gRPC authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.



	
  


 
:
   . The type of payload that should be returned.



 
(
   Compressable text format.


  

  
,
  Uncompressable binary format.


 

 
K
 > Randomly chosen from all other formats defined in this enum.


 

 
D
# (8 A block of data, to simply increase gRPC message size.



#
(
 % The type of data in body.


 %#

 %

 %

 %
+
' Primary contents of payload.


'%

'

'

'

+ < Unary request.



+
�
 . � Desired payload type in the response from the server.
 If response_type is RANDOM, server randomly chooses one from other formats.


 .+

 .

 .

 .
�
2� Desired payload size in the response from the server.
 If response_type is COMPRESSABLE, this denotes the size before compression.


2. 

2

2

2
B
55 Optional input payload sent along with the request.


52

5	

5


5
>
81 Whether SimpleResponse should include username.


85

8

8

8
A
;4 Whether SimpleResponse should include OAuth scope.


;8

;

;

;
;
? I/ Unary response, as configured by the request.



?
0
 A# Payload to increase message size.


 A?

 A	

 A


 A
x
Ek The user the request came from, for verifying authentication was
 successful when the client expected it.


EA

E

E	

E

H OAuth scope.


HE

H

H	

H
'
L Q Client-streaming request.



L!
B
 N5 Optional input payload sent along with the request.


 NL#

 N	

 N


 N
(
T W Client-streaming response.



T"
D
 V$7 Aggregated size of payloads received from the client.


 VT$

 V

 V

 V"#
6
Z b* Configuration for a particular response.



Z
�
 ]� Desired payload sizes in responses from the server.
 If response_type is COMPRESSABLE, this denotes the size before compression.


 ]Z

 ]

 ]

 ]
f
aY Desired interval between consecutive responses in the response stream in
 microseconds.


a]

a

a

a
'
e q Server-streaming request.



e"
�
 j � Desired payload type in the response from the server.
 If response_type is RANDOM, the payload from each response in the stream
 might be of different types. This is to simulate a mixed type of payload
 stream.


 je$

 j

 j

 j
@
m63 Configuration for each expected response message.


m


m

m1

m45
B
p5 Optional input payload sent along with the request.


pm6

p	

p


p
U
t wI Server-streaming response, as configured by the request and parameters.



t#
1
 v$ Payload to increase response size.


 vt%

 v	

 v


 v
�
 { �t A simple service to test the various types of RPCs and experiment with
 performance with various types of payload.



 {
@
  }'3 One empty request followed by one empty response.


  }

  }

  } %
c
 �8U One request followed by one response.
 The server returns the client payload as-is.


 �

 �

 �(6
�
 ��3� One request followed by a sequence of responses (streamed download).
 The server returns the payload with client desired type and sizes.


 �

 �4

 �

 �1
�
 ��+� A sequence of requests followed by one response (streamed upload).
 The server returns the aggregated size of client payload as the result.


 �

 �

 � 9

 �)
�
 ��3� A sequence of requests with each request served by the server immediately.
 As one request could lead to multiple responses, this interface
 demonstrates the idea of full duplexing.


 �

 �

 �6

 �

 �1
�
 ��3� A sequence of requests followed by a sequence of responses.
 The server buffers all the client requests and then serves them in order. A
 stream of responses are returned to the client when the server starts with
 first request.


 �

 �

 �6

 �

 �1bproto3