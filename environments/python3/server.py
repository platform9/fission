#!/usr/bin/env python

import logging
import sys
import imp
import os

from flask import Flask, request, abort, g

app = Flask(__name__)

userfunc = None

@app.route('/specialize', methods=['POST'])
def load():
    global userfunc
    # load user function from codepath
    codepath = '/userfunc/user'
    userfunc = (imp.load_source('user', codepath)).main
    return ""

@app.route('/v2/specialize', methods=['POST'])
def loadv2():
    global userfunc
    body = request.get_json()
    filepath = body['filepath']
    handler = body['functionName']

    # The value of "functionName" is consist of `<module-name>.<function-name>`.
    moduleName, funcName = handler.split(".")

    # check whether the destination is a directory or a file
    if os.path.isdir(filepath):
        # add package directory path into module search path
        sys.path.append(filepath)
        
        # find module from package path we append previously.
        # Python will try to find module from the same name file under
        # the package directory. If search is successful, the return 
        # value is a 3-element tuple; otherwise, an exception "ImportError"
        # is raised.
        fn, path, desc = imp.find_module(moduleName)

        # load module
        # Return module object is the load is successful; otherwise, 
        # an exception is raised.
        mod = imp.load_module(moduleName, fn, path, desc)
    else:
        # load source from destination python file
        mod = imp.load_source(moduleName, filepath)

    # load user function from module
    userfunc = getattr(mod, funcName)

    return ""

@app.route('/', methods=['GET', 'POST', 'PUT', 'HEAD', 'OPTIONS', 'DELETE'])
def f():
    if userfunc == None:
        print("Generic container: no requests supported")
        abort(500)
    #
    # Customizing the request context
    #
    # If you want to pass something to the function, you can add it to 'g':
    #   g.myKey = myValue
    # And the user func can then access that (after doing a "from flask import g").
    #
    return userfunc()

#
# Logging setup.  TODO: Loglevel hard-coded for now. We could allow
# functions/routes to override this somehow; or we could create
# separate dev vs. prod environments.
#
def setup_logger(loglevel):
    global app
    root = logging.getLogger()
    root.setLevel(loglevel)
    ch = logging.StreamHandler(sys.stdout)
    ch.setLevel(loglevel)
    ch.setFormatter(logging.Formatter('%(asctime)s - %(levelname)s - %(message)s'))
    app.logger.addHandler(ch)

#
# TODO: this starts the built-in server, which isn't the most
# efficient.  We should use something better.
#
setup_logger(logging.DEBUG)
app.logger.info("Starting server")
app.run(host='0.0.0.0', port='8888')
