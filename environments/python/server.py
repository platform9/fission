#!/usr/bin/env python

import logging
import sys
import imp
import os
import bjoern
from gevent.pywsgi import WSGIServer
from flask import Flask, request, abort, g

class FuncApp(Flask):
    def __init__(self, name, loglevel = logging.DEBUG):
        super(FuncApp, self).__init__(name)

        # init the class members
        self.userfunc = None
        self.root = logging.getLogger()
        self.ch = logging.StreamHandler(sys.stdout)

        #
        # Logging setup.  TODO: Loglevel hard-coded for now. We could allow
        # functions/routes to override this somehow; or we could create
        # separate dev vs. prod environments.
        #
        self.root.setLevel(loglevel)
        self.ch.setLevel(loglevel)
        self.ch.setFormatter(logging.Formatter('%(asctime)s - %(levelname)s - %(message)s'))
        self.logger.addHandler(self.ch)

        #
        # Register the routers
        #
        @self.route('/specialize', methods=['POST'])
        def load():
            # load user function from codepath
            codepath = '/self.userfunc/user'
            self.userfunc = (imp.load_source('user', codepath)).main
            return ""

        @self.route('/v2/specialize', methods=['POST'])
        def loadv2():
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
                # Second parameter of find_module enforces python to find same 
                # name module from the given list of directories to prevent name
                # confliction with built-in modules.
                f, path, desc = imp.find_module(moduleName, [filepath])
        
                # load module
                # Return module object is the load is successful; otherwise, 
                # an exception is raised.
                try:
                    mod = imp.load_module(moduleName, f, path, desc)
                finally:
                    if f:
                        f.close()
            else:
                # load source from destination python file
                mod = imp.load_source(moduleName, filepath)
        
            # load user function from module
            self.userfunc = getattr(mod, funcName)
        
            return ""

        @self.route('/healthz', methods=['GET'])
        def healthz():
            return "", 200

        @self.route('/', methods=['GET', 'POST', 'PUT', 'HEAD', 'OPTIONS', 'DELETE'])
        def f():
            if self.userfunc == None:
                print("Generic container: no requests supported")
                abort(500)
            #
            # Customizing the request context
            #
            # If you want to pass something to the function, you can add it to 'g':
            #   g.myKey = myValue
            # And the user func can then access that (after doing a "from flask import g").
            #
            return self.userfunc()

app = FuncApp(__name__, logging.DEBUG)

app.logger.info("Starting server")

#
# TODO: this starts the built-in server, which isn't the most
# efficient.  We should use something better.
#
if os.environ.get("WSGI_FRAMEWORK") == "GEVENT":
    svc = WSGIServer(('0.0.0.0', 8888), app)
    svc.serve_forever()
else:
    bjoern.run(app, '0.0.0.0', 8888, reuse_port=True)
