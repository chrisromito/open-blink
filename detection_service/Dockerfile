ARG TORCH_BASE=torch-base:1.0
FROM ${TORCH_BASE}

# The enviroment variable ensures that the python output is set straight
# to the terminal with out buffering it first
ENV PYTHONUNBUFFERED 1

# Set the working directory to /app
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .
EXPOSE 8000
CMD ["python", "main.py"]

