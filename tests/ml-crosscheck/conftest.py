import sys
from pathlib import Path

SRC = Path(__file__).resolve().parent.parent.parent / "ml-crosscheck" / "src"
sys.path.insert(0, str(SRC))
