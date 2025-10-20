# Source 0: https://medium.com/backticks-tildes/how-to-dockerize-a-django-application-a42df0cb0a99
# Source 1: https://github.com/two-trick-pony-NL/Django_AWS_Lightsail_Template/blob/main/dockerfile

# The first instruction is what image we want to base our container on
# We Use an official Python runtime as a parent image
FROM torch-base:1.0

# The enviroment variable ensures that the python output is set straight
# to the terminal with out buffering it first
ENV PYTHONUNBUFFERED 1

# Set the working directory to /app
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .

CMD ["python", "main.py"]

