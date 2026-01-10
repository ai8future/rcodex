#!/usr/bin/env python3
"""
Weather Check Script using OpenWeatherMap API
"""

import os
import sys
from typing import Any, Dict

# Check for requests library
try:
    import requests
except ImportError:
    print("Error: The 'requests' library is not installed.", file=sys.stderr)
    print("Please install it with: pip install requests", file=sys.stderr)
    sys.exit(1)

BASE_URL = "https://api.openweathermap.org/data/2.5/weather"
REQUEST_TIMEOUT_SECONDS = 10
UNITS = "metric"


class WeatherAPIError(Exception):
    """Custom exception for weather API errors."""


def _parse_weather_response(response: "requests.Response") -> Dict[str, Any]:
    try:
        data = response.json()
    except ValueError as exc:
        raise WeatherAPIError("Received invalid JSON response from API") from exc
    if not isinstance(data, dict):
        raise WeatherAPIError("Received unexpected API response format")
    return data


def get_weather(city: str, api_key: str) -> Dict[str, Any]:
    """
    Fetch weather data for a given city from OpenWeatherMap API.

    Args:
        city: Name of the city to check weather for
        api_key: OpenWeatherMap API key

    Returns:
        Dictionary containing weather data

    Raises:
        WeatherAPIError: If the API request fails
    """
    params = {"q": city, "appid": api_key, "units": UNITS}

    try:
        response = requests.get(BASE_URL, params=params, timeout=REQUEST_TIMEOUT_SECONDS)
        response.raise_for_status()
    except requests.exceptions.Timeout as exc:
        raise WeatherAPIError(f"Request timed out while fetching weather for {city}") from exc
    except requests.exceptions.HTTPError as exc:
        status_code = exc.response.status_code if exc.response is not None else 0
        if status_code == 401:
            raise WeatherAPIError("Invalid API key") from exc
        if status_code == 404:
            raise WeatherAPIError(f"City '{city}' not found") from exc
        if status_code == 429:
            raise WeatherAPIError("API rate limit exceeded") from exc
        raise WeatherAPIError(f"HTTP error occurred (Status: {status_code})") from exc
    except requests.exceptions.RequestException as exc:
        raise WeatherAPIError("Network error occurred while connecting to the API") from exc

    return _parse_weather_response(response)


def format_weather_data(data: Dict[str, Any]) -> str:
    """
    Format weather data into a readable string.

    Args:
        data: Weather data dictionary from API

    Returns:
        Formatted weather information string
    """
    city = data.get("name", "Unknown")
    country = data.get("sys", {}).get("country", "")
    location = f"{city}, {country}" if country else city
    temp = data.get("main", {}).get("temp")
    feels_like = data.get("main", {}).get("feels_like")
    humidity = data.get("main", {}).get("humidity")
    wind_speed = data.get("wind", {}).get("speed")

    weather_list = data.get("weather")
    if isinstance(weather_list, list) and weather_list:
        description = weather_list[0].get("description") or "No description"
    else:
        description = "No description"

    def format_value(value: Any, suffix: str = "") -> str:
        if isinstance(value, (int, float)):
            return f"{value}{suffix}"
        return "N/A"

    lines = [
        f"Weather for {location}:",
        "-----------------------------",
        f"Temperature: {format_value(temp, ' C')} (feels like {format_value(feels_like, ' C')})",
        f"Conditions: {description.capitalize()}",
        f"Humidity: {format_value(humidity, '%')}",
        f"Wind Speed: {format_value(wind_speed, ' m/s')}",
    ]
    return "\n".join(lines)


def main():
    """Main function to run the weather checker."""
    # Get API key from environment variable
    api_key = os.getenv("OPENWEATHER_API_KEY")

    if not api_key:
        print("Error: OPENWEATHER_API_KEY environment variable not set", file=sys.stderr)
        print("Please set it with: export OPENWEATHER_API_KEY='your_api_key'", file=sys.stderr)
        sys.exit(1)

    # Get city from command line argument or use default
    if len(sys.argv) > 1:
        city = " ".join(sys.argv[1:]).strip()
    else:
        print("Usage: python weather_check.py <city_name>")
        print("Example: python weather_check.py London")
        sys.exit(1)

    if not city:
        print("Error: City name cannot be empty", file=sys.stderr)
        sys.exit(1)

    try:
        weather_data = get_weather(city, api_key)
        print(format_weather_data(weather_data))
    except WeatherAPIError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception:
        print("Unexpected error occurred", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
