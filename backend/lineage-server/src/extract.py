from sqlparse import format,parse
import re
from typing import Optional
from sqlparse.sql import Token,Identifier

def remove_sql_comment(value:str)->str:
    return format(value,strip_comments=True).strip()

def extract_view(value:str)->Optional[str]:
    """
    Remove the create view definition and just return code
    """

    pattern = re.compile(r"CREATE\s+VIEW\s+(\[.*?\]|\S+)\s+AS\s*(.*)",re.IGNORECASE | re.DOTALL)

    match = pattern.search(value)

    if not match:
        return None
    
    view_name = match.group(1)

    sql_query = match.group(2).strip()

    #something view can be created with wrapped with () , so we have to remove them 
    
    if len(sql_query)>1 and sql_query.startswith("(") and sql_query.endswith(")"):
        sql_query = sql_query[1:-1].strip()

    return sql_query

def get_last_sql_token(sql:str)->Optional[Token]:
    """
    Return the last non-white space token in the sql or None on error
    """
    parsed = parse(sql)

    if parsed is None or len(parsed)==0:
        return None
    
    for token in reversed(parsed[0].tokens):
        if not token.is_whitespace:
            return token
        
    return None


def is_procedure_end_token(token:Optional[Token])->bool:
    """
    Return whether it is the store procedure end token such as END or GO
    """
    if token is None:
        return False
    
    
    #if the token is the identifier , it mean the procedure is complete and it is parsed as identifier

    return token.normalized == "END" or \
        token.normalized == "GO" or \
        isinstance(token,Identifier)


def extract_procedure(sql:str)->Optional[str]:
    """
    Extract the code part of store procedure
    """
    
    last_token = get_last_sql_token(sql=sql)

    if last_token is None:
        return None
    
    # the regex require the statment to be end with END or OR token. So we add the END if it does not have END token

    if not is_procedure_end_token(last_token):
        sql = sql+" END"

    pattern = re.compile(r'CREATE\s+PROC(?:EDURE)?\s+.*?AS\s+(?:BEGIN\s+)?(.*?)(?:(?!\bEND\b|\bGO\b).)*$', re.DOTALL | re.IGNORECASE)

    match = re.search(pattern, sql)

    if not match:
        return None
    
    sp_code = match.group(1)

    #the regex incorrectly match the last character so we have to remove it

    return sp_code[0:-1].strip()