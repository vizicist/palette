#ifndef _COXDEBOOR_H
#define _COXDEBOOR_H

class CoxDeBoorAlgorithm {

public:
	CoxDeBoorAlgorithm() {
		g_Points.clear();
	}

	void addPoint(NosuchVector pt) {
		g_Points.push_back(pt);
	}

	void endOfPoints() {

		g_num_cvs = g_Points.size();
		g_degree = 3;
		g_order=g_degree+1;
		g_num_knots=g_num_cvs+g_order;

		g_Knots.clear();
		g_Knots.push_back(0.0f);
		g_Knots.push_back(0.0f);
		g_Knots.push_back(0.0f);
		g_Knots.push_back(0.0f);
		for ( unsigned int k=1; k<=(g_num_cvs-g_order); k++ ) {
			g_Knots.push_back((float)k);
		}
		float lastval = (float)(g_num_cvs - g_degree);
		g_Knots.push_back(lastval);
		g_Knots.push_back(lastval);
		g_Knots.push_back(lastval);
		g_Knots.push_back(lastval);
	}

	float last_knot() {
		return g_Knots[g_num_knots-1];
	}

	float CoxDeBoor(float u,int i,int k,std::vector<float> Knots) {
		if(k==1)
		{
			if( Knots[i] <= u && u <= Knots[i+1] ) {
				return 1.0f;
			}
			return 0.0f;
		}
		float Den1 = Knots[i+k-1] - Knots[i];
		float Den2 = Knots[i+k] - Knots[i+1];
		float Eq1=0,Eq2=0;
		if(Den1>0) {
			Eq1 = ((u-Knots[i]) / Den1) * CoxDeBoor(u,i,k-1,Knots);
		}
		if(Den2>0) {
			Eq2 = (Knots[i+k]-u) / Den2 * CoxDeBoor(u,i+1,k-1,Knots);
		}
		return Eq1+Eq2;
	}
	
	NosuchVector GetOutpoint(float t) {
	
		// sum the effect of all CV's on the curve at this point to 
		// get the evaluated curve point
		// 
		NosuchVector outpt(0.0f,0.0f);
		for(unsigned int i=0;i!=g_num_cvs;++i) {
	
			// calculate the effect of this point on the curve
			float Val = CoxDeBoor(t,i,g_order,g_Knots);
	
			if(Val>0.001f) {
	
				// sum effect of CV on this part of the curve
				outpt.x += Val * g_Points[i].x;
				outpt.y += Val * g_Points[i].y;
			} else {
				NosuchDebug(2,"Val<=0.001f  : %f",Val);
			}
		}
		return outpt;
	}

private:
	std::vector<NosuchVector> g_Points;
	std::vector<float> g_Knots;
	unsigned int g_num_cvs;
	unsigned int g_degree;
	unsigned int g_order;
	unsigned int g_num_knots;
};

#endif