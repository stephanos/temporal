"""Read the original native owner's full GMSTORE v3 crash catalogue."""
import hashlib
import json
import struct


def catalogue(raw, log, volume):
    if len(raw)<56 or len(raw)>256<<20 or raw[:8]!=b"GMSTORE\x01":
        raise ValueError("native storage header/capacity")
    end=len(raw)-40
    if struct.unpack_from("<Q",raw,end)[0]!=end or hashlib.sha256(raw[:end]).digest()!=raw[end+8:]:
        raise ValueError("native storage integrity")
    if struct.unpack_from("<II",raw,8)!=(3,6):
        raise ValueError("native storage version/context")
    sections={}
    offset=16
    for expected in range(1,7):
        tag,size=struct.unpack_from("<IQ",raw,offset)
        offset+=12
        if tag!=expected or offset+size>end:
            raise ValueError("native storage section order/capacity")
        sections[tag]=raw[offset:offset+size]
        offset+=size
    if offset!=end or sections[5]!=struct.pack("<Q",len(log))+hashlib.sha256(log).digest():
        raise ValueError("native storage log binding")
    enumerations=json.loads(sections[3])
    contexts=json.loads(sections[6])
    if set(enumerations)!={volume} or set(contexts)!={volume}:
        raise ValueError("native storage volume mismatch")
    enumeration,context=enumerations[volume],contexts[volume]
    states=enumeration["States"]
    if not enumeration["Complete"] or enumeration["Capacity"] or enumeration["ChoiceExploration"] is not None or not states or len(states)>128:
        raise ValueError("incomplete native crash catalogue")
    if any(state["Volume"]!=volume or state["PendingSHA256"]!=context["PendingSHA256"] for state in states):
        raise ValueError("native crash context mismatch")
    if len({state["Identity"] for state in states})!=len(states):
        raise ValueError("duplicate native crash state")
    return enumeration,context
