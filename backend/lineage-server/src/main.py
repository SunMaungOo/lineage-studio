import grpc
from concurrent import futures 
from lineage_server_pb2 import LineageRequest,LineageResponse,ObjectType,Lineage,ObjectLineage
from lineage_server_pb2_grpc import LineageServiceServicer,add_LineageServiceServicer_to_server
from extract import remove_sql_comment,extract_view,extract_procedure
import sys
from sqlglot import parse
from lineage import find_parseable_ast,find_source_target_table
from typing import List,Set

def get_object_lineage(sources:Set[str],targets:List[str])->List[Lineage]:

    lineages:List[Lineage] = list()

    if len(sources)==0:
        return list()

    for target in targets:
        lineages.append(Lineage(
            target=target,
            source=sources
        ))

    return lineages

class LineageServer(LineageServiceServicer):
    def GetLineage(self, request:LineageRequest, context)->LineageResponse:

        object_lineages:List[ObjectLineage] = list()

        for object in request.objects:

            processed_sql = object.sql.strip()

            if len(processed_sql)==0:
                continue
            
            processed_sql = remove_sql_comment(value=processed_sql)

            if object.object_type==ObjectType.OBJECT_TYPE_VIEW:

                processed_sql = extract_view(value=processed_sql)

            elif object.object_type==ObjectType.OBJECT_TYPE_PROCEDURE:

                processed_sql = extract_procedure(value=processed_sql)

            else:
                continue            
        
            asts = parse(sql=processed_sql,dialect="tsql")

            lineages:List[Lineage] = list()

            for ast in find_parseable_ast(asts=asts):
                for sources,targets in find_source_target_table(ast=ast):
                    lineages.extend(get_object_lineage(sources=sources,targets=targets))

            if len(lineages)==0:
                continue

            object_lineages.append(ObjectLineage(
                name=object.name,
                lineages=lineages
            ))
                
        return LineageResponse(lineages=object_lineages)


def print_help():
    print("Arguments : <port> = the port (0-65535) to listen on")

def main(argv)->int:

    if len(argv)!=1:
        print_help()
        return -1

    port = -1

    try:
        port = int(argv[0])
    except ValueError:
        print("please use int for port")
        return -1
    
    if not(port>=0 and port <= 65535):
        print("please use valid port number")
        return -1
    
    server = grpc.server(thread_pool=futures.ThreadPoolExecutor(max_workers=10))

    add_LineageServiceServicer_to_server(LineageServer(),server)

    server.add_insecure_port(f"[::]:{port}")

    print(f"Listening on [::]:{port}")

    server.start()

    server.wait_for_termination()

    return 0

if __name__=="__main__":
    sys.exit(main(sys.argv[1:]))