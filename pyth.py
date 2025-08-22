import os
from dotenv import load_dotenv
from openai import OpenAI

# Load variables from .env file
load_dotenv()

# Now you can access them
api_key = os.getenv('OPENAI_API_KEY')
client = OpenAI(api_key=api_key) # Or just OpenAI() after load_dotenv()

print("API Key loaded successfully!")