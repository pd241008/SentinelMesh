import numpy as np
import torch
import torch.nn as nn
import torch.optim as optim

from dataset import FlowDataset

from .base import BaseScorer


class _Autoencoder(nn.Module):
    def __init__(self, input_dim: int, latent_dim: int = 2):
        super().__init__()
        self.encoder = nn.Sequential(
            nn.Linear(input_dim, 4),
            nn.ReLU(),
            nn.Linear(4, latent_dim),
        )
        self.decoder = nn.Sequential(
            nn.Linear(latent_dim, 4),
            nn.ReLU(),
            nn.Linear(4, input_dim),
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.decoder(self.encoder(x))


class AutoencoderScorer(BaseScorer):
    def __init__(
        self,
        latent_dim: int = 2,
        learning_rate: float = 1e-3,
        epochs: int = 100,
        batch_size: int = 32,
        device: str | None = None,
    ):
        self.latent_dim = latent_dim
        self.learning_rate = learning_rate
        self.epochs = epochs
        self.batch_size = batch_size
        self.device = device or ("cuda" if torch.cuda.is_available() else "cpu")
        self._model: _Autoencoder | None = None

    def train(self, dataset: FlowDataset) -> None:
        X = dataset.get_features(dataset.normal_flows)
        input_dim = X.shape[1]
        self._model = _Autoencoder(input_dim, self.latent_dim).to(self.device)
        criterion = nn.MSELoss()
        optimizer = optim.Adam(self._model.parameters(), lr=self.learning_rate)

        tensor_x = torch.tensor(X, dtype=torch.float32, device=self.device)
        loader = torch.utils.data.DataLoader(
            tensor_x, batch_size=self.batch_size, shuffle=True
        )

        self._model.train()
        for _ in range(self.epochs):
            for batch in loader:
                optimizer.zero_grad()
                output = self._model(batch)
                loss = criterion(output, batch)
                loss.backward()
                optimizer.step()

    def score(self, dataset: FlowDataset) -> np.ndarray:
        if self._model is None:
            raise RuntimeError("model must be trained before scoring")
        X = dataset.get_features()
        tensor_x = torch.tensor(X, dtype=torch.float32, device=self.device)
        self._model.eval()
        with torch.no_grad():
            reconstructed = self._model(tensor_x)
            mse = nn.functional.mse_loss(reconstructed, tensor_x, reduction="none")
            scores = mse.mean(dim=1).cpu().numpy()
        mn, mx = scores.min(), scores.max()
        if mx - mn < 1e-12:
            return np.full_like(scores, 0.0)
        return (scores - mn) / (mx - mn)
