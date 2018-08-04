#!/bin/bash

set -euo pipefail

ROOT=$(readlink -f ./../../../..)

for framework in python_bjoern python_gevent
do
	for executorType in poolmgr newdeploy
	do
	    for concurrency in 100 250 500 750 1000
	    do

	        testDuration="60"
	        dirName="concurrency-${concurrency}-executor-${executorType}"

	        # remove old data
	        rm -rf ${dirName}
	        mkdir ${dirName}
	        pushd ${dirName}

	        # run multiple iterations to reduce impact of imbalance of pod distribution.
	        for iteration in {1..10}
	        do

	            # Create a hello world function in nodejs, test it with an http trigger
	            echo "Pre-test cleanup"
	            fission env delete --name python || true

	            echo "Creating python env"
	            # Use short grace period time to speed up resource recycle time
	            # Use high min/max CPU so that K8S will distribute pod in different nodes
	            fission spec apply --specdir $ROOT/test/benchmark/assets/envs/$framework
	            trap "fission env delete --name python" EXIT

	            sleep 30

	            fn=python-hello-$(date +%s)

	            echo "Creating package"
	            rm -rf pkg.zip pkg/ || true
	            mkdir pkg
	            cp $ROOT/test/benchmark/assets/hello.py pkg/hello.py

	            zip -jr pkg.zip pkg/
	            pkgName=$(fission pkg create --env python --deploy pkg.zip | cut -d' ' -f 2 | cut -d"'" -f 2)

	            echo "Creating function"
	            fission fn create --name $fn --env python --pkg ${pkgName} --entrypoint "hello.main" --executortype ${executorType} --minscale 3 --maxscale 3

	            echo "Creating route"
	            fission route create --function $fn --url /$fn --method GET

	            echo "Waiting for router to catch up"
	            sleep 5

	            fnEndpoint="http://$FISSION_ROUTER/$fn"
	            js="sample.js"
	            rawFile="raw-${iteration}.json"
	            rawUsageReport="raw-usage.txt"

	            k6 run \
	                -e FN_ENDPOINT="${fnEndpoint}" \
	                --duration "${testDuration}s" \
	                --rps ${concurrency} \
	                --vus ${concurrency} \
	                --no-connection-reuse \
	                --out json="${rawFile}" \
	                --summary-trend-stats="avg,min,med,max,p(5),p(10),p(15),p(20),p(25),p(30)" \
	                ../${js} >> ${rawUsageReport}
	
	            echo "Clean up"
	            fission fn delete --name ${fn}
	            fission env delete --name python
	            fission route list| grep ${fn}| awk '{print $1}'| xargs fission route delete --name
	            fission pkg delete --name ${pkgName}
	            rm -rf pkg.zip pkg

	            kubectl -n fission-function get deploy -o name|xargs -I@ bash -c "kubectl -n fission-function delete @" || true
	            kubectl -n fission-function get pod -o name|xargs -I@ bash -c "kubectl -n fission-function delete @" || true

	            echo "All done."
	        done

	        usageReport="usage.txt"
	        cat ${rawUsageReport}| grep "http_req_duration"| cut -f2 -d':' > ${usageReport}

	        popd

	        # generate report after iterations are over
	        outImage="${dirName}.png"
	        picasso -file ${dirName} -format png -o ${outImage}

	    done
	done
done
