#!/usr/bin/env python3
"""
Convert GSI (Geospatial Information Authority of Japan) elevation data to binary format.
This script processes XML elevation data and creates a binary grid file for fast lookups.
"""

import struct
import xml.etree.ElementTree as ET
import numpy as np
from pathlib import Path
import argparse
import logging
from typing import Tuple, Optional
import glob
import re

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class GSIElevationConverter:
    def __init__(self, 
                 min_lat: float = 20.0,
                 max_lat: float = 46.0,
                 min_lon: float = 122.0,
                 max_lon: float = 154.0,
                 grid_size: float = 0.001):
        """
        Initialize the converter with grid parameters.
        
        Args:
            min_lat: Minimum latitude (default: 20.0)
            max_lat: Maximum latitude (default: 46.0)
            min_lon: Minimum longitude (default: 122.0)
            max_lon: Maximum longitude (default: 154.0)
            grid_size: Grid resolution in degrees (default: 0.001, ~100m)
        """
        self.min_lat = min_lat
        self.max_lat = max_lat
        self.min_lon = min_lon
        self.max_lon = max_lon
        self.grid_size = grid_size
        
        self.width = int((max_lon - min_lon) / grid_size)
        self.height = int((max_lat - min_lat) / grid_size)
        
        self.grid = np.full((self.height, self.width), -9999, dtype=np.int16)
        
        logger.info(f"Grid dimensions: {self.width} x {self.height}")
        logger.info(f"Grid size: {grid_size} degrees (~{grid_size * 111000:.0f}m)")
        logger.info(f"Coverage: Lat {min_lat}-{max_lat}, Lon {min_lon}-{max_lon}")
    
    def parse_gsi_xml(self, xml_path: str) -> None:
        """
        Parse GSI XML elevation data file.
        
        Args:
            xml_path: Path to the GSI XML file
        """
        try:
            tree = ET.parse(xml_path)
            root = tree.getroot()
            
            ns = {'gml': 'http://www.opengis.net/gml/3.2'}
            
            for tuple_list in root.findall('.//gml:tupleList', ns):
                text = tuple_list.text.strip()
                for line in text.split('\n'):
                    parts = line.strip().split(',')
                    if len(parts) >= 3:
                        try:
                            lat = float(parts[0])
                            lon = float(parts[1])
                            elev = float(parts[2])
                            
                            self.set_elevation(lat, lon, elev)
                        except ValueError:
                            continue
            
            logger.info(f"Processed XML file: {xml_path}")
            
        except Exception as e:
            logger.error(f"Error processing {xml_path}: {e}")
    
    def parse_csv_data(self, csv_path: str) -> None:
        """
        Parse CSV elevation data file.
        
        Args:
            csv_path: Path to the CSV file
        """
        try:
            with open(csv_path, 'r') as f:
                for line in f:
                    line = line.strip()
                    if not line or line.startswith('#'):
                        continue
                    
                    parts = line.split(',')
                    if len(parts) >= 3:
                        try:
                            lat = float(parts[0])
                            lon = float(parts[1])
                            elev = float(parts[2])
                            
                            self.set_elevation(lat, lon, elev)
                        except ValueError:
                            continue
            
            logger.info(f"Processed CSV file: {csv_path}")
            
        except Exception as e:
            logger.error(f"Error processing {csv_path}: {e}")
    
    def set_elevation(self, lat: float, lon: float, elevation: float) -> None:
        """
        Set elevation for a given coordinate.
        
        Args:
            lat: Latitude
            lon: Longitude
            elevation: Elevation in meters
        """
        if lat < self.min_lat or lat > self.max_lat:
            return
        if lon < self.min_lon or lon > self.max_lon:
            return
        
        x = int((lon - self.min_lon) / self.grid_size)
        y = int((lat - self.min_lat) / self.grid_size)
        
        if 0 <= x < self.width and 0 <= y < self.height:
            elevation_cm = int(elevation * 100)
            
            elevation_cm = max(-32768, min(32767, elevation_cm))
            
            self.grid[y, x] = elevation_cm
    
    def interpolate_missing(self) -> None:
        """
        Interpolate missing data points using nearest neighbor interpolation.
        """
        logger.info("Interpolating missing data points...")
        
        from scipy import ndimage
        
        mask = self.grid != -9999
        
        indices = ndimage.distance_transform_edt(~mask, return_distances=False, return_indices=True)
        
        self.grid = self.grid[tuple(indices)]
        
        logger.info("Interpolation complete")
    
    def save_binary(self, output_path: str) -> None:
        """
        Save the grid as a binary file.
        
        Args:
            output_path: Path for the output binary file
        """
        with open(output_path, 'wb') as f:
            self.grid.astype('<i2').tofile(f)
        
        logger.info(f"Saved binary data to {output_path}")
        logger.info(f"File size: {Path(output_path).stat().st_size / (1024**2):.2f} MB")
    
    def save_header(self, header_path: str) -> None:
        """
        Save the grid header information.
        
        Args:
            header_path: Path for the header file
        """
        with open(header_path, 'wb') as f:
            f.write(struct.pack('<i', self.width))
            f.write(struct.pack('<i', self.height))
            f.write(struct.pack('<d', self.min_lat))
            f.write(struct.pack('<d', self.max_lat))
            f.write(struct.pack('<d', self.min_lon))
            f.write(struct.pack('<d', self.max_lon))
            f.write(struct.pack('<d', self.grid_size))
        
        logger.info(f"Saved header to {header_path}")
    
    def generate_test_data(self) -> None:
        """
        Generate test elevation data for development.
        """
        logger.info("Generating test elevation data...")
        
        for y in range(self.height):
            for x in range(self.width):
                lat = self.min_lat + y * self.grid_size
                lon = self.min_lon + x * self.grid_size
                
                if 35.36 <= lat <= 35.37 and 138.72 <= lon <= 138.73:
                    self.grid[y, x] = 377600
                elif 35.68 <= lat <= 35.69 and 139.76 <= lon <= 139.77:
                    self.grid[y, x] = 300
                elif 34.68 <= lat <= 34.69 and 135.52 <= lon <= 135.53:
                    self.grid[y, x] = 2000
                else:
                    base_elevation = 500 + x * 0.01 + y * 0.02
                    self.grid[y, x] = int(base_elevation)
        
        logger.info("Test data generation complete")
    
    def get_statistics(self) -> dict:
        """
        Get statistics about the elevation data.
        
        Returns:
            Dictionary with statistics
        """
        valid_data = self.grid[self.grid != -9999]
        
        if len(valid_data) == 0:
            return {
                'total_points': self.width * self.height,
                'valid_points': 0,
                'missing_points': self.width * self.height,
                'coverage': 0.0
            }
        
        return {
            'total_points': self.width * self.height,
            'valid_points': len(valid_data),
            'missing_points': np.sum(self.grid == -9999),
            'min_elevation': float(np.min(valid_data) / 100),
            'max_elevation': float(np.max(valid_data) / 100),
            'mean_elevation': float(np.mean(valid_data) / 100),
            'coverage': len(valid_data) / (self.width * self.height) * 100
        }

def main():
    parser = argparse.ArgumentParser(description='Convert GSI elevation data to binary format')
    parser.add_argument('--input', '-i', type=str, help='Input file or directory path')
    parser.add_argument('--output', '-o', type=str, default='data/elevation.bin',
                        help='Output binary file path (default: data/elevation.bin)')
    parser.add_argument('--header', type=str, default='data/elevation.bin.header',
                        help='Output header file path (default: data/elevation.bin.header)')
    parser.add_argument('--test', action='store_true',
                        help='Generate test data instead of processing input files')
    parser.add_argument('--interpolate', action='store_true',
                        help='Interpolate missing data points')
    parser.add_argument('--min-lat', type=float, default=20.0,
                        help='Minimum latitude (default: 20.0)')
    parser.add_argument('--max-lat', type=float, default=46.0,
                        help='Maximum latitude (default: 46.0)')
    parser.add_argument('--min-lon', type=float, default=122.0,
                        help='Minimum longitude (default: 122.0)')
    parser.add_argument('--max-lon', type=float, default=154.0,
                        help='Maximum longitude (default: 154.0)')
    parser.add_argument('--grid-size', type=float, default=0.001,
                        help='Grid size in degrees (default: 0.001)')
    
    args = parser.parse_args()
    
    Path(args.output).parent.mkdir(parents=True, exist_ok=True)
    Path(args.header).parent.mkdir(parents=True, exist_ok=True)
    
    converter = GSIElevationConverter(
        min_lat=args.min_lat,
        max_lat=args.max_lat,
        min_lon=args.min_lon,
        max_lon=args.max_lon,
        grid_size=args.grid_size
    )
    
    if args.test:
        converter.generate_test_data()
    elif args.input:
        input_path = Path(args.input)
        
        if input_path.is_dir():
            xml_files = list(input_path.glob('**/*.xml'))
            csv_files = list(input_path.glob('**/*.csv'))
            
            logger.info(f"Found {len(xml_files)} XML files and {len(csv_files)} CSV files")
            
            for xml_file in xml_files:
                converter.parse_gsi_xml(str(xml_file))
            
            for csv_file in csv_files:
                converter.parse_csv_data(str(csv_file))
        
        elif input_path.suffix.lower() == '.xml':
            converter.parse_gsi_xml(str(input_path))
        
        elif input_path.suffix.lower() == '.csv':
            converter.parse_csv_data(str(input_path))
        
        else:
            logger.error(f"Unsupported file type: {input_path.suffix}")
            return
        
        if args.interpolate:
            converter.interpolate_missing()
    else:
        logger.error("Please specify --input or --test option")
        return
    
    converter.save_binary(args.output)
    converter.save_header(args.header)
    
    stats = converter.get_statistics()
    logger.info("Data statistics:")
    for key, value in stats.items():
        if isinstance(value, float):
            logger.info(f"  {key}: {value:.2f}")
        else:
            logger.info(f"  {key}: {value}")

if __name__ == '__main__':
    main()