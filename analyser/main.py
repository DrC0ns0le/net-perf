import os
import sys

sys.path.append(os.path.dirname(os.path.abspath(__file__)))
import grpc
import time
import numpy as np
from scipy.optimize import lsq_linear
import logging
from proto.networkanalysis import networkanalysis_pb2_grpc, networkanalysis_pb2
from concurrent import futures

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MatrixSolver(networkanalysis_pb2_grpc.NetworkAnalyzerServicer):
    def SolveMatrix(self, request, context):
        try:
            logger.info("Received matrix solving request")
            
            # start time
            start_time = time.time()
            
            # Reshape the matrix and vector
            A = np.array(request.matrix).reshape(request.matrix_rows, request.matrix_cols)
            b = np.array(request.vector)
            
            logger.info(f"Matrix shape: {A.shape}, Vector shape: {b.shape}")
            
            # Solve using lsq_linear
            result = lsq_linear(
                A, b,
                method='bvls', 
                tol=1e-8,    
                lsq_solver='exact', 
                max_iter=10000,
            )
            
            logger.info("Successfully solved matrix")
            
            # end time in ms
            end_time = time.time()
            logger.info(f"Time taken to solve matrix: {(end_time - start_time) * 1000} ms")
            
            return networkanalysis_pb2.SolutionResponse(
                solution=result.x.tolist(),
                success=True,
                error=""
            )
        except Exception as e:
            logger.error(f"Error solving matrix: {str(e)}")
            return networkanalysis_pb2.SolutionResponse(
                solution=[],
                success=False,
                error=str(e)
            )

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    networkanalysis_pb2_grpc.add_NetworkAnalyzerServicer_to_server(
        MatrixSolver(), server)
    server.add_insecure_port('[::]:50052')
    server.start()
    logger.info("Python solver server started on port 50052")
    server.wait_for_termination()

if __name__ == '__main__':
    serve()