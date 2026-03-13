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
    duration: str = "2 mins"
    style: str = "conversational"
    language: str = "english"

class ScriptResponse(BaseModel):
    script: str

def generate_llm_script(summary: str, context: Dict[str, str], duration: str, style: str, language: str) -> str:
    """Calls Sarvam API to generate a highly realistic podcast script."""
    
    context_str = "\n".join([f"- {name}: {text}" for name, text in context.items()])
    
    # Language instruction
    language_instruction = "The conversation must be in English."
    if language.lower() == "malayalam":
        language_instruction = "The entire conversation MUST be in Malayalam (using Malayalam script)."

    prompt = f"""
You are the lead scriptwriter for 'Foundry', a top-tier podcast. Your goal is to write a script that sounds like a real-life, unscripted conversation between two close friends.

PERSONAS:
- Host A (Sameer): A curious, slightly skeptical tech-enthusiast. He likes to ask "But wait, how does that actually work?" or say "Honestly, that sounds wild." He's the one who sets the scene.
- Host B (Anjali): An energetic, brilliant expert who loves deep-dives. She gets excited about small details and uses phrases like "Exactly!", "You won't believe this," or "Actually, there's a catch."

REQUIREMENTS:
1. LANGUAGE: {language_instruction}
2. DURATION: {duration}
3. STYLE: {style} (ensure the energy matches this)

TOPIC & CONTEXT:
Summary: {summary}
Deep-dive nuggets: {context_str}

REALISM GUIDELINES:
- Use natural filler words: "um", "uh", "like", "you know", "honestly", "actually".
- Include verbal interjections: Host B should occasionally react mid-sentence when Host A says something surprising (e.g., "Right!", "Wow").
- Pacing: Use ellipses (...) or dashes (--) for natural pauses and thinking moments.

MANDATORY INTRO:
- The conversation MUST start with a professional, high-energy intro.
- Host A (Sameer) and Host B (Anjali) must introduce themselves and the show "Foundry".
- Example: "Hey everyone, Sameer here!" "And I'm Anjali, welcome back to Foundry. Today we're diving into..."
- This must be the very first part of the conversation.

MANDATORY OUTRO:
- The conversation MUST end with a formal sign-off.
- Host A (Sameer) MUST say: "Thanks for listening to Foundry, I'm Sameer."
- Host B (Anjali) MUST say: "And I'm Anjali. See you in the next one!"
- These must be the very last two lines of the transcript.

STRICT FORMATTING:
- Every single line MUST start with exactly "Host A: " or "Host B: ".
- Host A is ALWAYS Sameer (Male).
- Host B is ALWAYS Anjali (Female).
- Do NOT include any meta-text, stage directions, or role descriptions.
"""

    headers = {
        "Content-Type": "application/json",
        "api-subscription-key": SARVAM_API_KEY
    }
    
    payload = {
        "model": "sarvam-30b",
        "messages": [
            {"role": "system", "content": "You are a professional podcast scriptwriter specialized in high-energy, natural Indian-context conversations."},
            {"role": "user", "content": prompt}
        ],
        "temperature": 0.7 # Stabilized temperature
    }

    try:
        response = requests.post(SARVAM_URL, json=payload, headers=headers, timeout=60)
        if response.status_code != 200:
            print(f"Sarvam API Error: {response.status_code} - {response.text}")
        response.raise_for_status()
        data = response.json()
        script = data['choices'][0]['message']['content']
        print("\n--- GENERATED SCRIPT ---")
        print(script)
        print("------------------------\n")
        return script
    except Exception as e:
        print(f"Exception during Sarvam LLM call: {e}")
        # Fallback
        if language.lower() == "malayalam":
            return f"Host A: ക്ഷമിക്കണം, സ്ക്രിപ്റ്റ് നിർമ്മിക്കുന്നതിൽ പിശക് സംഭവിച്ചു. വിഷയം: {summary[:50]}\nHost B: ദയവായി നിങ്ങളുടെ API കീ പരിശോധിക്കുക."
        return f"Host A: Data fetch failed, but we're talking about: {summary[:50]}...\nHost B: And we had context from {len(context)} sources. Check your API key!"

@app.post("/generate-script", response_model=ScriptResponse)
async def generate_script(request: ScriptRequest):
    script_content = generate_llm_script(
        request.summary, 
        request.context, 
        request.duration, 
        request.style, 
        request.language
    )
    return ScriptResponse(script=script_content)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)