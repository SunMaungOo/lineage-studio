FROM python:3.12-alpine

WORKDIR /app

COPY backend/lineage-server/requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt

COPY backend/lineage-server/src/ ./src/

ENTRYPOINT ["python","src/main.py"]