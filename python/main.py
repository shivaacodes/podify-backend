from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import Dict
import requests
import os

app = FastAPI()

# Sarvam API Key
SARVAM_API_KEY = "YOUR_SARVAM_API_KEY_HERE"
SARVAM_URL = "https://api.sarvam.ai/v1/chat/completions"

class ScriptRequest(BaseModel):
    summary: str
    context: Dict[str, str]

class ScriptResponse(BaseModel):
    script: str

def generate_llm_script(summary: str, context: Dict[str, str]) -> str:
    """Calls Sarvam API to generate a podcast script."""
    
    context_str = "\n".join([f"- {name}: {text}" for name, text in context.items()])
    
    prompt = f"""
You are an expert podcast script writer. Create a natural, engaging conversation between two hosts, Host A and Host B.

MAIN TOPIC SUMMARY:
{summary}

ADDITIONAL CONTEXT NUGGETS:
{context_str}

STRICT GUIDELINES:
1. Format each line as "Host A: [text]" or "Host B: [text]".
2. Make it sound like a real, enthusiastic conversation, not a clinical summary.
3. Host A is curious and sets the stage. Host B is the expert who brings in the context nuggets.
4. Keep it under 2 minutes of speaking time.
5. Do not include any stage directions like [Laughter] or [Intro Music]. Just the dialogue.
"""

    headers = {
        "Content-Type": "application/json",
        "api-subscription-key": SARVAM_API_KEY
    }
    
    payload = {
        "model": "sarvam-30b", # Correct Sarvam chat model
        "messages": [
            {"role": "system", "content": "You are a professional podcast scriptwriter."},
            {"role": "user", "content": prompt}
        ],
        "temperature": 0.7
    }

    try:
        response = requests.post(SARVAM_URL, json=payload, headers=headers, timeout=30)
        if response.status_code != 200:
            print(f"Sarvam API Error: {response.status_code} - {response.text}")
        response.raise_for_status()
        data = response.json()
        return data['choices'][0]['message']['content']
    except Exception as e:
        print(f"Exception during Sarvam LLM call: {e}")
        # Fallback to a semi-dynamic stub if API fails
        return f"Host A: Data fetch failed, but we're talking about: {summary[:50]}...\nHost B: And we had context from {len(context)} sources. Check your API key!"

@app.post("/generate-script", response_model=ScriptResponse)
async def generate_script(request: ScriptRequest):
    script_content = generate_llm_script(request.summary, request.context)
    return ScriptResponse(script=script_content)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)