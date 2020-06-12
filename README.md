logsync is a terminal based utility for viewing multiple log files from a distributed 
system such that their timestamps are in sync as you step through the logs.
To run logsync, type:
logsync file1 file2 ... filen

Once it loads, there will be 4 text boxes for each log file specified and a command edit box.
Here are the valid commands:
    "head" jumps all files to the beginning
    "tail" jumps all files to the end
    Any positive number jumps that many steps, where each step chooses the next
    log line based on time stamp and advancing that file foward one.
    Any negative number goes back that many steps.
    Also it is possible to search based on a timestamp like "2020-05-25|08:47:33.663" to jump to the closest log line for all the files

    To build:
    make build
    This will create the logsync executable in the current directory, and you can put it in a folder that 
    is in your path or just run it where it is.

    To test out the logsync utility, you can use the sample log files under test_data and run like this:
    "./logsync test_data/logs/*"
    This will run the logsync against the 4 log files that were generated from a Tendermint cluster.


    To clean:
    make clean

    To test the filechunk code to make sure its valid:
    make test

