CoinSpark libraries README - http://coinspark.org/

ABOUT
-----
This is an unofficial port of the CoinSpark library to Go.

The CoinSpark libraries help you integrate support for the CoinSpark protocol into
your wallet, or any other tool or service.

For more information and code examples: http://coinspark.org/developers/

HOW TO INCLUDE THE LIBRARY
--------------------------

* From the command-line:

go get github.com/bitcartel/go-coinspark/coinspark

HOW TO TEST
-----------

* Compile the Go test tool:

cd coinspark-test
go build
./coinspark-test

* Prepare some standard tests by following the instructions here: https://github.com/coinspark/libraries

* Navigate to the directory containing the test files:

cd CoinSpark-Tests-*

* For each '...-Input.txt' file, run the coinspark-test tool on that input:

coinspark-test Address-Input.txt > Address-Output-GO.txt
coinspark-test AssetRef-Input.txt > AssetRef-Output-GO.txt
coinspark-test Script-Input.txt > Script-Output-GO.txt
coinspark-test AssetHash-Input.txt > AssetHash-Output-GO.txt
coinspark-test Genesis-Input.txt > Genesis-Output-GO.txt
coinspark-test Transfer-Input.txt > Transfer-Output-GO.txt
coinspark-test MessageHash-Input.txt > MessageHash-Output-GO.txt

* Now check the corresponding GO and C output files for differences:

diff Address-Output-C.txt Address-Output-GO.txt
diff AssetRef-Output-C.txt AssetRef-Output-GO.txt
diff Script-Output-C.txt Script-Output-GO.txt
diff AssetHash-Output-C.txt AssetHash-Output-GO.txt
diff Genesis-Output-C.txt Genesis-Output-GO.txt
diff Transfer-Output-C.txt Transfer-Output-GO.txt
diff MessageHash-Output-C.txt MessageHash-Output-GO.txt

* If no differences were reported, the library has passed the C-GO consistency test.

* Feel free to look inside the input and output files to see what is going on.


LICENSE (MIT)
-------------

Copyright (c) Simon Liu

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.



CHANGELOG
---------

v2.1 - 3 April 2015
- First release of the Go port

