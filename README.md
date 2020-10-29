Steps to Run the application

1. Direct Runner<br/>
    `go run app.go --output wordcounts.txt --runner direct`

2. Run using Spark Runner

    - Start Spark (If using binaries to start local spark - Use 2.4.7 not 3.x, then skip this step)
        - Use below command to start locally in distributed mode using docker
            - Start Master<br/>
                `docker run --name spark-master -h spark-master -e ENABLE_INIT_DAEMON=false -d bde2020/spark-master:3.0.0-hadoop3.2`
            - Start Worker<br/>
                `docker run --name spark-worker-1 --link spark-master:spark-master -e ENABLE_INIT_DAEMON=false -d bde2020/spark-worker:3.0.0-hadoop3.2`
        - Check connection using <br/>
            - `spark-shell --master spark://localhost:7077`
            - Run this code in terminal (It should return PI)
                ```
                val NUM_SAMPLES=10000
                    var count = sc.parallelize(1 to NUM_SAMPLES).filter { _ =>
                    val x = math.random
                    val y = math.random
                    x*x + y*y < 1
                    }.count() * 4/(NUM_SAMPLES.toFloat)
                ```
            - Check application status at http://localhost:4040
            - Check Master status at http://localhost:8080
            - Check Worker status at http://localhost:8081
    - Start Beam JobServer<br/>
        `docker run --net=host apache/beam_spark_job_server:latest --spark-master-url=spark://192.168.1.11:7077`
        - If running manually
            `java -jar ~/Downloads/beam-runners-spark-job-server-2.25.0.jar --spark-master-url=spark://192.168.1.11:7077 --job-port=8099 --artifact-port 0`
    - Run Application using SparkRunner<br/>
        `go run app.go -output=wordcounts.txt -runner=spark -endpoint=localhost:8099 --environment_type=LOOPBACK`

3. 