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
    functionName = body['functionName']
    # add filepath into syspath for module import
    sys.path.append(filepath)
    fn, path, desc = imp.find_module('user', [filepath])
    mod = imp.load_module('user', fn, path, desc)
    userfunc = getattr(mod, functionName)
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
