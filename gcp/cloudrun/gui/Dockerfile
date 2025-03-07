# Use official Python runtime
FROM python:3.12

# Set the working directory
WORKDIR /app

# Copy application files
COPY . .

# Install dependencies
RUN pip install --upgrade pip
RUN pip install --no-cache-dir -r requirements.txt

# Set Flask environment variable
ENV PORT=8080

# Expose the port Flask will run on
EXPOSE 8080

# Run Flask app
CMD ["python3", "main.py"]